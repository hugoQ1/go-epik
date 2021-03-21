package flowch

import (
	"context"

	"golang.org/x/xerrors"

	"github.com/ipfs/go-cid"
	"go.uber.org/fx"

	"github.com/filecoin-project/go-address"

	"github.com/EpiK-Protocol/go-epik/api"
	"github.com/EpiK-Protocol/go-epik/chain/actors/builtin/flowch"
	"github.com/EpiK-Protocol/go-epik/chain/types"
	"github.com/EpiK-Protocol/go-epik/flowchmgr"
)

type FlowchAPI struct {
	fx.In

	FlowchMgr *flowchmgr.Manager
}

func (a *FlowchAPI) FlowchGet(ctx context.Context, from, to address.Address, amt types.BigInt) (*api.ChannelInfo, error) {
	ch, mcid, err := a.FlowchMgr.GetFlowch(ctx, from, to, amt)
	if err != nil {
		return nil, err
	}

	return &api.ChannelInfo{
		Channel:      ch,
		WaitSentinel: mcid,
	}, nil
}

func (a *FlowchAPI) FlowchAvailableFunds(ctx context.Context, ch address.Address) (*api.ChannelAvailableFunds, error) {
	return a.FlowchMgr.AvailableFunds(ch)
}

func (a *FlowchAPI) FlowchAvailableFundsByFromTo(ctx context.Context, from, to address.Address) (*api.ChannelAvailableFunds, error) {
	return a.FlowchMgr.AvailableFundsByFromTo(from, to)
}

func (a *FlowchAPI) FlowchGetWaitReady(ctx context.Context, sentinel cid.Cid) (address.Address, error) {
	return a.FlowchMgr.GetFlowchWaitReady(ctx, sentinel)
}

func (a *FlowchAPI) FlowchAllocateLane(ctx context.Context, ch address.Address) (uint64, error) {
	return a.FlowchMgr.AllocateLane(ch)
}

func (a *FlowchAPI) FlowchNewPayment(ctx context.Context, from, to address.Address, vouchers []api.FlowVoucherSpec) (*api.FlowInfo, error) {
	amount := vouchers[len(vouchers)-1].Amount

	// TODO: Fix free fund tracking in FlowchGet
	// TODO: validate voucher spec before locking funds
	ch, err := a.FlowchGet(ctx, from, to, amount)
	if err != nil {
		return nil, err
	}

	lane, err := a.FlowchMgr.AllocateLane(ch.Channel)
	if err != nil {
		return nil, err
	}

	svs := make([]*flowch.SignedVoucher, len(vouchers))

	for i, v := range vouchers {
		sv, err := a.FlowchMgr.CreateVoucher(ctx, ch.Channel, flowch.SignedVoucher{
			Amount: v.Amount,
			Lane:   lane,

			Extra:           v.Extra,
			TimeLockMin:     v.TimeLockMin,
			TimeLockMax:     v.TimeLockMax,
			MinSettleHeight: v.MinSettle,
		})
		if err != nil {
			return nil, err
		}
		if sv.Voucher == nil {
			return nil, xerrors.Errorf("Could not create voucher - shortfall of %d", sv.Shortfall)
		}

		svs[i] = sv.Voucher
	}

	return &api.FlowInfo{
		Channel:      ch.Channel,
		WaitSentinel: ch.WaitSentinel,
		Vouchers:     svs,
	}, nil
}

func (a *FlowchAPI) FlowchList(ctx context.Context) ([]address.Address, error) {
	return a.FlowchMgr.ListChannels()
}

func (a *FlowchAPI) FlowchStatus(ctx context.Context, pch address.Address) (*api.FlowchStatus, error) {
	ci, err := a.FlowchMgr.GetChannelInfo(pch)
	if err != nil {
		return nil, err
	}
	return &api.FlowchStatus{
		ControlAddr: ci.Control,
		Direction:   api.PCHDir(ci.Direction),
	}, nil
}

func (a *FlowchAPI) FlowchSettle(ctx context.Context, addr address.Address) (cid.Cid, error) {
	return a.FlowchMgr.Settle(ctx, addr)
}

func (a *FlowchAPI) FlowchCollect(ctx context.Context, addr address.Address) (cid.Cid, error) {
	return a.FlowchMgr.Collect(ctx, addr)
}

func (a *FlowchAPI) FlowchVoucherCheckValid(ctx context.Context, ch address.Address, sv *flowch.SignedVoucher) error {
	return a.FlowchMgr.CheckVoucherValid(ctx, ch, sv)
}

func (a *FlowchAPI) FlowchVoucherCheckSpendable(ctx context.Context, ch address.Address, sv *flowch.SignedVoucher, secret []byte, proof []byte) (bool, error) {
	return a.FlowchMgr.CheckVoucherSpendable(ctx, ch, sv, secret, proof)
}

func (a *FlowchAPI) FlowchVoucherAdd(ctx context.Context, ch address.Address, sv *flowch.SignedVoucher, proof []byte, minDelta types.BigInt) (types.BigInt, error) {
	return a.FlowchMgr.AddVoucherInbound(ctx, ch, sv, proof, minDelta)
}

// FlowchVoucherCreate creates a new signed voucher on the given payment channel
// with the given lane and amount.  The value passed in is exactly the value
// that will be used to create the voucher, so if previous vouchers exist, the
// actual additional value of this voucher will only be the difference between
// the two.
// If there are insufficient funds in the channel to create the voucher,
// returns a nil voucher and the shortfall.
func (a *FlowchAPI) FlowchVoucherCreate(ctx context.Context, pch address.Address, amt types.BigInt, lane uint64) (*api.FlowVoucherCreateResult, error) {
	return a.FlowchMgr.CreateVoucher(ctx, pch, flowch.SignedVoucher{Amount: amt, Lane: lane})
}

func (a *FlowchAPI) FlowchVoucherList(ctx context.Context, pch address.Address) ([]*flowch.SignedVoucher, error) {
	vi, err := a.FlowchMgr.ListVouchers(ctx, pch)
	if err != nil {
		return nil, err
	}

	out := make([]*flowch.SignedVoucher, len(vi))
	for k, v := range vi {
		out[k] = v.Voucher
	}

	return out, nil
}

func (a *FlowchAPI) FlowchVoucherSubmit(ctx context.Context, ch address.Address, sv *flowch.SignedVoucher, secret []byte, proof []byte) (cid.Cid, error) {
	return a.FlowchMgr.SubmitVoucher(ctx, ch, sv, secret, proof)
}
