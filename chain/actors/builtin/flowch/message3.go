package flowch

import (
	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"

	builtin3 "github.com/filecoin-project/specs-actors/v2/actors/builtin"
	flowch3 "github.com/filecoin-project/specs-actors/v2/actors/builtin/flowch"
	init3 "github.com/filecoin-project/specs-actors/v2/actors/builtin/init"

	"github.com/EpiK-Protocol/go-epik/chain/actors"
	init_ "github.com/EpiK-Protocol/go-epik/chain/actors/builtin/init"
	"github.com/EpiK-Protocol/go-epik/chain/types"
)

type message3 struct{ from address.Address }

func (m message3) Create(to address.Address, initialAmount abi.TokenAmount) (*types.Message, error) {
	params, aerr := actors.SerializeParams(&flowch3.ConstructorParams{From: m.from, To: to, Amount: initialAmount})
	if aerr != nil {
		return nil, aerr
	}
	enc, aerr := actors.SerializeParams(&init3.ExecParams{
		CodeCID:           builtin3.FlowChannelActorCodeID,
		ConstructorParams: params,
	})
	if aerr != nil {
		return nil, aerr
	}

	return &types.Message{
		To:     init_.Address,
		From:   m.from,
		Value:  abi.NewTokenAmount(0),
		Method: builtin3.MethodsInit.Exec,
		Params: enc,
	}, nil
}

func (m message3) AddFunds(flowch address.Address, amount abi.TokenAmount) (*types.Message, error) {
	params, aerr := actors.SerializeParams(&flowch3.AddFundsParams{Amount: amount})
	if aerr != nil {
		return nil, aerr
	}

	return &types.Message{
		To:     flowch,
		From:   m.from,
		Value:  abi.NewTokenAmount(0),
		Method: builtin3.MethodsFlowch.AddFunds,
		Params: params,
	}, nil
}

func (m message3) Update(flowch address.Address, sv *SignedVoucher, secret []byte) (*types.Message, error) {
	params, aerr := actors.SerializeParams(&flowch3.UpdateChannelStateParams{
		Sv:     *sv,
		Secret: secret,
	})
	if aerr != nil {
		return nil, aerr
	}

	return &types.Message{
		To:     flowch,
		From:   m.from,
		Value:  abi.NewTokenAmount(0),
		Method: builtin3.MethodsFlowch.UpdateChannelState,
		Params: params,
	}, nil
}

func (m message3) Settle(flowch address.Address) (*types.Message, error) {
	return &types.Message{
		To:     flowch,
		From:   m.from,
		Value:  abi.NewTokenAmount(0),
		Method: builtin3.MethodsFlowch.Settle,
	}, nil
}

func (m message3) Collect(flowch address.Address) (*types.Message, error) {
	return &types.Message{
		To:     flowch,
		From:   m.from,
		Value:  abi.NewTokenAmount(0),
		Method: builtin3.MethodsFlowch.Collect,
	}, nil
}
