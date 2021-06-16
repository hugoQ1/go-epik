package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"

	"github.com/EpiK-Protocol/go-epik/api"
	"github.com/EpiK-Protocol/go-epik/chain/stmgr"
	types "github.com/EpiK-Protocol/go-epik/chain/types"
	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-jsonrpc"
	"github.com/filecoin-project/go-state-types/abi"
	cid "github.com/ipfs/go-cid"
	cbg "github.com/whyrusleeping/cbor-gen"
	"golang.org/x/xerrors"
)

//go:generate go run github.com/golang/mock/mockgen -destination=servicesmock_test.go -package=cli -self_package github.com/EpiK-Protocol/go-epik/cli . ServicesAPI

type ServicesAPI interface {
	API() api.FullNode
	// Sends executes a send given SendParams
	Send(ctx context.Context, params SendParams) (cid.Cid, error)
	// BatchSends executes a send given SendParams
	BatchSend(ctx context.Context, params []SendParams) ([]cid.Cid, error)
	// DecodeTypedParamsFromJSON takes in information needed to identify a method and converts JSON
	// parameters to bytes of their CBOR encoding
	DecodeTypedParamsFromJSON(ctx context.Context, to address.Address, method abi.MethodNum, paramstr string) ([]byte, error)

	// Close ends the session of services and disconnects from RPC, using Services after Close is called
	// most likely will result in an error
	// Should not be called concurrently
	Close() error
}

type ServicesImpl struct {
	api    api.FullNode
	closer jsonrpc.ClientCloser
}

func (s *ServicesImpl) Close() error {
	if s.closer == nil {
		return xerrors.Errorf("Services already closed")
	}
	s.closer()
	s.closer = nil
	return nil
}

func (s *ServicesImpl) DecodeTypedParamsFromJSON(ctx context.Context, to address.Address, method abi.MethodNum, paramstr string) ([]byte, error) {
	act, err := s.api.StateGetActor(ctx, to, types.EmptyTSK)
	if err != nil {
		return nil, err
	}

	methodMeta, found := stmgr.MethodsMap[act.Code][method]
	if !found {
		return nil, fmt.Errorf("method %d not found on actor %s", method, act.Code)
	}

	p := reflect.New(methodMeta.Params.Elem()).Interface().(cbg.CBORMarshaler)

	if err := json.Unmarshal([]byte(paramstr), p); err != nil {
		return nil, fmt.Errorf("unmarshaling input into params type: %w", err)
	}

	buf := new(bytes.Buffer)
	if err := p.MarshalCBOR(buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

type SendParams struct {
	To   address.Address
	From address.Address
	Val  abi.TokenAmount

	GasPremium *abi.TokenAmount
	GasFeeCap  *abi.TokenAmount
	GasLimit   *int64

	Nonce  *uint64
	Method abi.MethodNum
	Params []byte

	Force bool
}

// This is specialised Send for Send command
// There might be room for generic Send that other commands can use to send their messages
// We will see

var ErrSendBalanceTooLow = errors.New("balance too low")

func (s *ServicesImpl) API() api.FullNode {
	return s.api
}

func (s *ServicesImpl) Send(ctx context.Context, params SendParams) (cid.Cid, error) {
	msg, err := s.parseMessage(ctx, params)
	if err != nil {
		return cid.Undef, err
	}

	if params.Nonce != nil {
		msg.Nonce = *params.Nonce
		sm, err := s.api.WalletSignMessage(ctx, params.From, msg)
		if err != nil {
			return cid.Undef, err
		}

		_, err = s.api.MpoolPush(ctx, sm)
		if err != nil {
			return cid.Undef, err
		}

		return sm.Cid(), nil
	}

	sm, err := s.api.MpoolPushMessage(ctx, msg, nil)
	if err != nil {
		return cid.Undef, err
	}

	return sm.Cid(), nil
}

func (s *ServicesImpl) BatchSend(ctx context.Context, params []SendParams) ([]cid.Cid, error) {
	var (
		msgs  []*types.Message
		smsgs []*types.SignedMessage
		cids  []cid.Cid
	)
	for _, p := range params {
		msg, err := s.parseMessage(ctx, p)
		if err != nil {
			return nil, err
		}

		if p.Nonce != nil {
			msg.Nonce = *p.Nonce
			sm, err := s.api.WalletSignMessage(ctx, p.From, msg)
			if err != nil {
				return nil, err
			}
			smsgs = append(smsgs, sm)
		} else {
			msgs = append(msgs, msg)
		}
	}

	if len(smsgs) > 0 {
		ids, err := s.api.MpoolBatchPush(ctx, smsgs)
		if err != nil {
			return nil, err
		}
		cids = append(cids, ids...)
	}

	if len(msgs) > 0 {
		smsgs, err := s.api.MpoolBatchPushMessage(ctx, msgs, nil)
		if err != nil {
			return nil, err
		}
		for _, sm := range smsgs {
			cids = append(cids, sm.Cid())
		}
	}
	return cids, nil
}

func (s *ServicesImpl) parseMessage(ctx context.Context, params SendParams) (*types.Message, error) {
	if params.From == address.Undef {
		defaddr, err := s.api.WalletDefaultAddress(ctx)
		if err != nil {
			return nil, err
		}
		params.From = defaddr
	}

	msg := &types.Message{
		From:  params.From,
		To:    params.To,
		Value: params.Val,

		Method: params.Method,
		Params: params.Params,
	}

	if params.GasPremium != nil {
		msg.GasPremium = *params.GasPremium
	} else {
		msg.GasPremium = types.NewInt(0)
	}
	if params.GasFeeCap != nil {
		msg.GasFeeCap = *params.GasFeeCap
	} else {
		msg.GasFeeCap = types.NewInt(0)
	}
	if params.GasLimit != nil {
		msg.GasLimit = *params.GasLimit
	} else {
		msg.GasLimit = 0
	}

	if !params.Force {
		// Funds insufficient check
		fromBalance, err := s.api.WalletBalance(ctx, msg.From)
		if err != nil {
			return nil, err
		}
		totalCost := types.BigAdd(types.BigMul(msg.GasFeeCap, types.NewInt(uint64(msg.GasLimit))), msg.Value)

		if fromBalance.LessThan(totalCost) {
			return nil, xerrors.Errorf("From balance %s less than total cost %s: %w", types.EPK(fromBalance), types.EPK(totalCost), ErrSendBalanceTooLow)

		}
	}

	return msg, nil
}
