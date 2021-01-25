package retrievaladapter

import (
	"context"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-fil-markets/retrievalmarket"
	"github.com/filecoin-project/go-fil-markets/shared"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/ipfs/go-cid"
	"github.com/multiformats/go-multiaddr"
	"golang.org/x/xerrors"

	"github.com/EpiK-Protocol/go-epik/build"
	"github.com/EpiK-Protocol/go-epik/chain/actors"
	"github.com/EpiK-Protocol/go-epik/chain/actors/builtin/paych"
	"github.com/EpiK-Protocol/go-epik/chain/actors/builtin/retrieval"
	retrievalactor "github.com/EpiK-Protocol/go-epik/chain/actors/builtin/retrieval"
	"github.com/EpiK-Protocol/go-epik/chain/types"
	"github.com/EpiK-Protocol/go-epik/node/impl/full"
	payapi "github.com/EpiK-Protocol/go-epik/node/impl/paych"
)

type retrievalClientNode struct {
	chainAPI full.ChainAPI
	payAPI   payapi.PaychAPI
	stateAPI full.StateAPI
	mpoolAPI full.MpoolAPI
}

// NewRetrievalClientNode returns a new node adapter for a retrieval client that talks to the
// Epik Node
func NewRetrievalClientNode(payAPI payapi.PaychAPI, chainAPI full.ChainAPI, stateAPI full.StateAPI, mpoolAPI full.MpoolAPI) retrievalmarket.RetrievalClientNode {
	return &retrievalClientNode{payAPI: payAPI, chainAPI: chainAPI, stateAPI: stateAPI, mpoolAPI: mpoolAPI}
}

// SubmitDataPledge submit data pledge
func (rcn *retrievalClientNode) SubmitDataPledge(ctx context.Context, clientAddress, minerAddress address.Address, pieceCid cid.Cid, size uint64) (cid.Cid, error) {
	params, aerr := actors.SerializeParams(&retrievalactor.RetrievalData{
		PieceID:  pieceCid,
		Size:     size,
		Client:   clientAddress,
		Provider: clientAddress,
	})
	if aerr != nil {
		return cid.Undef, aerr
	}

	msg := types.Message{
		To:     retrievalactor.Address,
		From:   clientAddress,
		Value:  abi.NewTokenAmount(0),
		Method: retrievalactor.Methods.RetrievalData,
		Params: params,
	}
	sm, err := rcn.mpoolAPI.MpoolPushMessage(ctx, &msg, nil)
	if err != nil {
		return cid.Undef, err
	}
	return sm.Cid(), nil
}

// WaitForDataPledgeReady wait data pledge ready
func (rcn *retrievalClientNode) WaitForDataPledgeReady(ctx context.Context, waitSentinel cid.Cid) error {
	ret, err := rcn.stateAPI.StateWaitMsg(ctx, waitSentinel, build.MessageConfidence)

	if err != nil {
		return xerrors.Errorf("waiting for data pledge message: %w", err)
	}
	if ret.Receipt.ExitCode != 0 {
		return xerrors.Errorf("data pledge failed: exit=%d", ret.Receipt.ExitCode)
	}
	return nil
}

// ConfirmComplete confirm deal complete
func (rcn *retrievalClientNode) ConfirmComplete(ctx context.Context, clientAddress, minerAddress address.Address, pieceCid cid.Cid, size uint64) (cid.Cid, error) {
	params, aerr := actors.SerializeParams(&retrievalactor.RetrievalData{
		PieceID:  pieceCid,
		Size:     size,
		Client:   clientAddress,
		Provider: minerAddress,
	})
	if aerr != nil {
		return cid.Undef, aerr
	}

	msg := types.Message{
		To:     retrieval.Address,
		From:   clientAddress,
		Value:  abi.NewTokenAmount(0),
		Method: retrieval.Methods.ConfirmData,
		Params: params,
	}
	sm, err := rcn.mpoolAPI.MpoolPushMessage(ctx, &msg, nil)
	if err != nil {
		return cid.Undef, err
	}
	return sm.Cid(), nil
}

// GetOrCreatePaymentChannel sets up a new payment channel if one does not exist
// between a client and a miner and ensures the client has the given amount of
// funds available in the channel.
func (rcn *retrievalClientNode) GetOrCreatePaymentChannel(ctx context.Context, clientAddress address.Address, minerAddress address.Address, clientFundsAvailable abi.TokenAmount, tok shared.TipSetToken) (address.Address, cid.Cid, error) {
	// TODO: respect the provided TipSetToken (a serialized TipSetKey) when
	// querying the chain
	ci, err := rcn.payAPI.PaychGet(ctx, clientAddress, minerAddress, clientFundsAvailable)
	if err != nil {
		return address.Undef, cid.Undef, err
	}
	return ci.Channel, ci.WaitSentinel, nil
}

// Allocate late creates a lane within a payment channel so that calls to
// CreatePaymentVoucher will automatically make vouchers only for the difference
// in total
func (rcn *retrievalClientNode) AllocateLane(ctx context.Context, paymentChannel address.Address) (uint64, error) {
	return rcn.payAPI.PaychAllocateLane(ctx, paymentChannel)
}

// CreatePaymentVoucher creates a new payment voucher in the given lane for a
// given payment channel so that all the payment vouchers in the lane add up
// to the given amount (so the payment voucher will be for the difference)
func (rcn *retrievalClientNode) CreatePaymentVoucher(ctx context.Context, paymentChannel address.Address, amount abi.TokenAmount, lane uint64, tok shared.TipSetToken) (*paych.SignedVoucher, error) {
	// TODO: respect the provided TipSetToken (a serialized TipSetKey) when
	// querying the chain
	voucher, err := rcn.payAPI.PaychVoucherCreate(ctx, paymentChannel, amount, lane)
	if err != nil {
		return nil, err
	}
	if voucher.Voucher == nil {
		return nil, retrievalmarket.NewShortfallError(voucher.Shortfall)
	}
	return voucher.Voucher, nil
}

func (rcn *retrievalClientNode) GetChainHead(ctx context.Context) (shared.TipSetToken, abi.ChainEpoch, error) {
	head, err := rcn.chainAPI.ChainHead(ctx)
	if err != nil {
		return nil, 0, err
	}

	return head.Key().Bytes(), head.Height(), nil
}

func (rcn *retrievalClientNode) WaitForPaymentChannelReady(ctx context.Context, messageCID cid.Cid) (address.Address, error) {
	return rcn.payAPI.PaychGetWaitReady(ctx, messageCID)
}

func (rcn *retrievalClientNode) CheckAvailableFunds(ctx context.Context, paymentChannel address.Address) (retrievalmarket.ChannelAvailableFunds, error) {

	channelAvailableFunds, err := rcn.payAPI.PaychAvailableFunds(ctx, paymentChannel)
	if err != nil {
		return retrievalmarket.ChannelAvailableFunds{}, err
	}
	return retrievalmarket.ChannelAvailableFunds{
		ConfirmedAmt:        channelAvailableFunds.ConfirmedAmt,
		PendingAmt:          channelAvailableFunds.PendingAmt,
		PendingWaitSentinel: channelAvailableFunds.PendingWaitSentinel,
		QueuedAmt:           channelAvailableFunds.QueuedAmt,
		VoucherReedeemedAmt: channelAvailableFunds.VoucherReedeemedAmt,
	}, nil
}

func (rcn *retrievalClientNode) GetKnownAddresses(ctx context.Context, p retrievalmarket.RetrievalPeer, encodedTs shared.TipSetToken) ([]multiaddr.Multiaddr, error) {
	tsk, err := types.TipSetKeyFromBytes(encodedTs)
	if err != nil {
		return nil, err
	}
	mi, err := rcn.stateAPI.StateMinerInfo(ctx, p.Address, tsk)
	if err != nil {
		return nil, err
	}
	multiaddrs := make([]multiaddr.Multiaddr, 0, len(mi.Multiaddrs))
	for _, a := range mi.Multiaddrs {
		maddr, err := multiaddr.NewMultiaddrBytes(a)
		if err != nil {
			return nil, err
		}
		multiaddrs = append(multiaddrs, maddr)
	}

	return multiaddrs, nil
}
