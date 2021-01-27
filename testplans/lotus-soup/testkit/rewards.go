package testkit

import (
	"context"
	"fmt"

	"github.com/EpiK-Protocol/go-epik/api"
	"github.com/EpiK-Protocol/go-epik/build"
	"github.com/EpiK-Protocol/go-epik/chain/actors"
	"github.com/EpiK-Protocol/go-epik/chain/actors/builtin"
	"github.com/EpiK-Protocol/go-epik/chain/actors/builtin/retrieval"
	"github.com/EpiK-Protocol/go-epik/chain/actors/builtin/reward"
	"github.com/EpiK-Protocol/go-epik/chain/types"
	tstats "github.com/EpiK-Protocol/go-epik/tools/stats"
	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/big"
	builtin2 "github.com/filecoin-project/specs-actors/v2/actors/builtin"
	reward2 "github.com/filecoin-project/specs-actors/v2/actors/builtin/reward"
	"golang.org/x/xerrors"
)

func SendRetrievePledge(t *TestEnvironment, ctx context.Context, client api.FullNode) abi.ChainEpoch {

	pa := epkToAttoEpk(Float64Param(t, "pledge_amount"))
	if pa.IsZero() {
		t.RecordMessage("pledge amount is zero")
		return abi.ChainEpoch(-1)
	}

	from, err := client.WalletDefaultAddress(ctx)
	if err != nil {
		panic(err)
	}

	t.RecordMessage("send %s from %s for retrieval pledge", types.EPK(pa), from)

	sp, err := actors.SerializeParams(&from)
	if err != nil {
		panic(err)
	}

	msg, err := client.MpoolPushMessage(ctx, &types.Message{
		To:     retrieval.Address,
		From:   from,
		Value:  pa,
		Method: retrieval.Methods.AddBalance,
		Params: sp,
	}, nil)
	if err != nil {
		panic(err)
	}

	t.RecordMessage("send retrieve pledge message: %s", msg.Cid())

	// wait
	mlk, err := client.StateWaitMsg(ctx, msg.Cid(), 1)
	if err != nil {
		panic(err)
	}
	if mlk.Receipt.ExitCode != 0 {
		panic(fmt.Sprintf("message exit %d", mlk.Receipt.ExitCode))
	}

	rs, err := client.StateRetrievalPledge(ctx, from, types.EmptyTSK)
	if err != nil {
		panic(err)
	}

	t.RecordMessage("pledge at %d, amount %s, state (balance %s, dayexpend %s, locked %s, locked epoch %d)", mlk.Height, types.EPK(pa),
		types.EPK(rs.Balance), types.EPK(rs.DayExpend), types.EPK(rs.Locked), rs.LockedEpoch)

	return mlk.Height
}

func WithdrawRetrievePledge(t *TestEnvironment, ctx context.Context, client api.FullNode) abi.ChainEpoch {

	wpa := epkToAttoEpk(Float64Param(t, "withdraw_pledge_amount"))
	if wpa.IsZero() {
		t.RecordMessage("withdraw amount is zero")
		return abi.ChainEpoch(-1)
	}

	from, err := client.WalletDefaultAddress(ctx)
	if err != nil {
		panic(err)
	}

	t.RecordMessage("withdraw pledge %s", types.EPK(wpa))

	sp, err := actors.SerializeParams(&retrieval.WithdrawBalanceParams{
		ProviderOrClientAddress: from,
		Amount:                  wpa,
	})
	if err != nil {
		panic(err)
	}

	msg, err := client.MpoolPushMessage(ctx, &types.Message{
		To:     retrieval.Address,
		From:   from,
		Value:  big.Zero(),
		Method: retrieval.Methods.WithdrawBalance,
		Params: sp,
	}, nil)
	if err != nil {
		panic(err)
	}

	t.RecordMessage("send withdraw pledge message: %s", msg.Cid())

	// wait
	mlk, err := client.StateWaitMsg(ctx, msg.Cid(), 1)
	if err != nil {
		panic(err)
	}
	if mlk.Receipt.ExitCode != 0 {
		panic(fmt.Sprintf("message exit %d", mlk.Receipt.ExitCode))
	}

	rs, err := client.StateRetrievalPledge(ctx, from, types.EmptyTSK)
	if err != nil {
		panic(err)
	}

	t.RecordMessage("withdraw at %d, amount %s, state (balance %s, dayexpend %s, locked %s, locked epoch %d)", mlk.Height, types.EPK(wpa),
		types.EPK(rs.Balance), types.EPK(rs.DayExpend), types.EPK(rs.Locked), rs.LockedEpoch)

	return mlk.Height
}

func RunCheckRewards(t *TestEnvironment, ctx context.Context, genesisAccounts []address.Address, client api.FullNode, quit <-chan error) {

	f := func(addr address.Address, tsk types.TipSetKey) abi.TokenAmount {
		act, err := client.StateGetActor(ctx, addr, tsk)
		if err != nil {
			panic(err)
		}
		return act.Balance
	}

	// get init balance
	genTs, err := client.ChainGetGenesis(ctx)
	if err != nil {
		panic(err)
	}

	// sum genesis account balance
	genesisAccountsBalance := big.Zero()
	for _, ga := range genesisAccounts {
		genesisAccountsBalance = big.Add(genesisAccountsBalance, f(ga, genTs.Key()))
	}

	initTeam := f(builtin.TeamIDAddress, genTs.Key())
	initFoundation := f(builtin.FoundationIDAddress, genTs.Key())
	initFundraising := f(builtin.FundraisingIDAddress, genTs.Key())
	t.RecordMessage("genesis alloc balance: team %s, foundation %s, fundraising %s", types.EPK(initTeam), types.EPK(initFoundation), types.EPK(initFundraising))

	initCirc, err := client.StateVMCirculatingSupplyInternal(ctx, genTs.Key())
	if err != nil {
		panic(err)
	}
	t.RecordMessage("@%d vested %s, circ %s, pledge %s", genTs.Height(), types.EPK(initCirc.EpkVested), types.EPK(initCirc.EpkCirculating), types.EPK(initCirc.TotalRetrievalPledge))
	t.RecordMessage("genesis alloc vested: team %s, foundation %s, fundraising %s",
		types.EPK(initCirc.EpkTeamVested), types.EPK(initCirc.EpkFoundationVested), types.EPK(initCirc.EpkFundraisingVested))

	// Expect: genesis vested funds is zero
	RequireEquals(initCirc.EpkVested, big.Sum(initCirc.EpkTeamVested, initCirc.EpkFoundationVested, initCirc.EpkFundraisingVested), genTs, "genesis vested")

	height := 0
	headlag := 3

	cctx, cancel := context.WithCancel(ctx)
	defer cancel()

	tipsetsCh, err := tstats.GetTips(cctx, client, abi.ChainEpoch(height), headlag)
	if err != nil {
		panic(err)
	}

	var (
		lastCirc   *api.CirculatingSupply
		lastDetail *reward.TotalMinedDetail
	)

	for tipset := range tipsetsCh {
		select {
		case err := <-quit:
			t.RecordMessage("@%d checker stopped: %w", tipset.Height(), err)
			return
		default:
		}

		circ, err := client.StateVMCirculatingSupplyInternal(ctx, tipset.Key())
		if err != nil {
			panic(err)
		}
		t.RecordMessage("@%d vested %s, current: pledge %s, circ %s", tipset.Height(), types.EPK(circ.EpkVested), types.EPK(circ.TotalRetrievalPledge), types.EPK(circ.EpkCirculating))

		detail, err := client.StateTotalMinedDetail(ctx, tipset.Key())
		if err != nil {
			panic(err)
		}

		actualVested := big.Sum(circ.EpkTeamVested, circ.EpkFoundationVested, circ.EpkFundraisingVested)
		curTsBalance := big.Zero()
		for _, ga := range genesisAccounts {
			curTsBalance = big.Add(curTsBalance, f(ga, tipset.Key()))
		}
		if curTsBalance.LessThan(genesisAccountsBalance) {
			actualVested = big.Add(actualVested, big.Sub(genesisAccountsBalance, curTsBalance))
		}

		// Expect: total vested == alloc vested (team/foundation/fundraising) + genesis account paid out (clients & miners)
		RequireEquals(circ.EpkVested, actualVested, tipset, "total vested")
		if lastCirc != nil {
			RequireGreaterThan(circ.EpkVested, lastCirc.EpkVested, tipset, "total vested inc")
		}

		// Expect: mined vote rewards == fallback (foundation) gained + vote fund balance
		fallbackGained := big.Sub(f(builtin.FoundationIDAddress, tipset.Key()), initFoundation)
		voteFundBalance := f(builtin2.VoteFundActorAddr, tipset.Key())
		RequireEquals(big.Add(fallbackGained, voteFundBalance), detail.TotalVoteReward, tipset, "mined vote rewards")
		t.RecordMessage("@%d fallback gained %s, vote fund balance %s, mined vote rewards %s", tipset.Height(), types.EPK(fallbackGained), types.EPK(voteFundBalance), types.EPK(detail.TotalVoteReward))

		// Expect: total reward alloc == reward actor balance + mined
		rewardBal := f(builtin2.RewardActorAddr, tipset.Key())
		totalMined := big.Sum(detail.TotalExpertReward, detail.TotalKnowledgeReward, detail.TotalRetrievalReward, detail.TotalStoragePowerReward, detail.TotalVoteReward)
		RequireEquals(circ.EpkMined, totalMined, tipset, "total mined")
		RequireEquals(big.Add(rewardBal, totalMined), big.NewFromGo(build.InitialRewardBalance), tipset, "total reward alloc")
		t.RecordMessage("@%d total alloc %s, reward balance %s, mined %s", tipset.Height(), types.EPK(big.NewFromGo(build.InitialRewardBalance)), types.EPK(rewardBal), types.EPK(totalMined))

		// Expect: send to vote/knowledge/expert/retrieve/power
		if lastDetail != nil {
			deltaVote := big.Zero()
			deltaKnow := big.Zero()
			deltaExpert := big.Zero()
			deltaRetr := big.Zero()
			deltaPower := big.Zero()

			blockReward := big.Div(reward2.EpochZeroReward, big.NewInt(int64(len(tipset.Cids()))))
			hasRetr := false
			for i := 0; i < len(tipset.Cids()); i++ {
				vote := big.Div(blockReward, big.NewInt(100))                           // 1% to vote
				expert := big.Div(big.Mul(blockReward, big.NewInt(9)), big.NewInt(100)) // 9% to expert
				p15 := big.Div(big.Mul(blockReward, big.NewInt(15)), big.NewInt(100))   // 15% to knowledge & retrieval

				retr := big.Zero()
				// NOTE use lastCirc
				if !lastCirc.TotalRetrievalPledge.IsZero() && !lastCirc.EpkCirculating.IsZero() {
					hasRetr = true
					retr = big.Mul(lastCirc.TotalRetrievalPledge, blockReward)
					retr = big.Div(retr, big.Mul(lastCirc.EpkCirculating, big.NewInt(5)))
					t.RecordMessage("@%d block reward %s, parent: pledge %s, circulating %s, ", tipset.Height(), types.EPK(blockReward),
						types.EPK(lastCirc.TotalRetrievalPledge), types.EPK(lastCirc.EpkCirculating))
				}
				know := big.Sub(p15, retr)

				deltaVote = big.Add(deltaVote, vote)
				deltaExpert = big.Add(deltaExpert, expert)
				deltaRetr = big.Add(deltaRetr, retr)
				deltaKnow = big.Add(deltaKnow, know)
				deltaPower = big.Add(deltaPower, big.Subtract(blockReward, vote, expert, p15))
			}

			actualTotal := big.Sum(deltaVote, deltaKnow, deltaExpert, deltaRetr, deltaPower)

			t.RecordMessage("@%d award %s to: vote(%s), know(%s), expert(%s), retrieve(%t - %s), power(%s)",
				tipset.Height(),
				types.EPK(actualTotal),
				types.EPK(deltaVote),
				types.EPK(deltaKnow),
				types.EPK(deltaExpert),
				hasRetr,
				types.EPK(deltaRetr),
				types.EPK(deltaPower),
			)

			RequireEquals(big.Sub(circ.EpkMined, lastCirc.EpkMined), actualTotal, tipset, "delta total")
			RequireEquals(big.Sub(detail.TotalVoteReward, lastDetail.TotalVoteReward), deltaVote, tipset, "delta vote")
			RequireEquals(big.Sub(detail.TotalKnowledgeReward, lastDetail.TotalKnowledgeReward), deltaKnow, tipset, "delta konwledge")
			RequireEquals(big.Sub(detail.TotalExpertReward, lastDetail.TotalExpertReward), deltaExpert, tipset, "delta expert")
			RequireEquals(big.Sub(detail.TotalRetrievalReward, lastDetail.TotalRetrievalReward), deltaRetr, tipset, "delta retrieve")
			RequireEquals(big.Sub(detail.TotalStoragePowerReward, lastDetail.TotalStoragePowerReward), deltaPower, tipset, "delta power")
		}
		lastDetail = detail
		lastCirc = &circ

		t.RecordMessage("rewards check ok at %d", tipset.Height())
	}
}

func RequireEquals(a, b abi.TokenAmount, ts *types.TipSet, msg string) {
	if !a.Equals(b) {
		panic(xerrors.Errorf("@%d require '%s': %s == %s", ts.Height(), msg, types.EPK(a), types.EPK(b)))
	}
}

func RequireGreaterThan(a, b abi.TokenAmount, ts *types.TipSet, msg string) {
	if !a.GreaterThan(b) {
		panic(xerrors.Errorf("@%d require '%s': %s > %s", ts.Height(), msg, types.EPK(a), types.EPK(b)))
	}
}

func SleepEpochs(t *TestEnvironment, ctx context.Context, client api.FullNode, epochs int) {

	if epochs < 0 {
		panic("negative epochs not allowed")
	}

	cctx, cancel := context.WithCancel(ctx)
	defer cancel()

	head, err := client.ChainHead(ctx)
	if err != nil {
		panic(err)
	}

	headlag := 3
	tipsetsCh, err := tstats.GetTips(cctx, client, head.Height(), headlag)
	if err != nil {
		panic(err)
	}

	var tipset *types.TipSet
	for tipset = range tipsetsCh {
		if tipset.Height() >= head.Height()+abi.ChainEpoch(epochs) {
			break
		}
	}

	t.RecordMessage("sleep %d epochs from %d, final %d", epochs, head.Height(), tipset.Height())
}
