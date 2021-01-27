package testkit

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	"github.com/EpiK-Protocol/go-epik/api"
	"github.com/EpiK-Protocol/go-epik/chain/actors"
	"github.com/EpiK-Protocol/go-epik/chain/actors/builtin"
	"github.com/EpiK-Protocol/go-epik/chain/types"
	tstats "github.com/EpiK-Protocol/go-epik/tools/stats"
	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/big"
	builtin2 "github.com/filecoin-project/specs-actors/v2/actors/builtin"
	"github.com/filecoin-project/specs-actors/v2/actors/builtin/vote"
	"golang.org/x/xerrors"
)

func VoteSend(t *TestEnvironment, ctx context.Context, client api.FullNode, from, candidate address.Address) abi.ChainEpoch {
	var err error
	va := epkToAttoEpk(Float64Param(t, "vote_send_amount"))
	if va.IsZero() {
		t.RecordMessage("vote amount is zero")
		return abi.ChainEpoch(-1)
	}

	t.RecordMessage("vote %s from %s for %s", types.EPK(va), from, candidate)

	// send votes
	sp, err := actors.SerializeParams(&candidate)
	if err != nil {
		panic(err)
	}

	msg, err := client.MpoolPushMessage(ctx, &types.Message{
		To:     builtin2.VoteFundActorAddr,
		From:   from,
		Value:  va,
		Method: builtin2.MethodsVote.Vote,
		Params: sp,
	}, nil)
	if err != nil {
		panic(err)
	}

	t.RecordMessage("send vote message: %s", msg.Cid())

	// wait
	mlk, err := client.StateWaitMsg(ctx, msg.Cid(), 1)
	if err != nil {
		panic(err)
	}
	if mlk.Receipt.ExitCode != 0 {
		panic(fmt.Sprintf("message exit %d", mlk.Receipt.ExitCode))
	}

	info, err := client.StateVoterInfo(ctx, from, types.EmptyTSK)
	if err != nil {
		panic(err)
	}
	totalVotes := big.Zero()
	for _, cand := range info.Candidates {
		totalVotes = big.Add(totalVotes, cand)
	}
	t.RecordMessage("vote msg at %d, total %s, rewards %s, candidates %v", mlk.Height,
		types.EPK(totalVotes), types.EPK(info.WithdrawableRewards), convertCandidateList(info.Candidates))

	return mlk.Height
}

func VoteRescind(t *TestEnvironment, ctx context.Context, client api.FullNode, from, candidate address.Address) abi.ChainEpoch {
	var err error
	ra := epkToAttoEpk(Float64Param(t, "vote_rescind_amount"))
	if ra.IsZero() {
		t.RecordMessage("rescind amount is zero")
		return abi.ChainEpoch(-1)
	}

	t.RecordMessage("rescind %s from %s of %s", types.EPK(ra), candidate, from)

	// rescind votes
	sp, err := actors.SerializeParams(&vote.RescindParams{
		Candidate: candidate,
		Votes:     types.BigInt(ra),
	})
	if err != nil {
		panic(err)
	}

	msg, err := client.MpoolPushMessage(ctx, &types.Message{
		To:     builtin2.VoteFundActorAddr,
		From:   from,
		Value:  big.Zero(),
		Method: builtin2.MethodsVote.Rescind,
		Params: sp,
	}, nil)
	if err != nil {
		panic(err)
	}

	t.RecordMessage("send rescind message: %s", msg.Cid())

	// wait
	mlk, err := client.StateWaitMsg(ctx, msg.Cid(), 1)
	if err != nil {
		panic(err)
	}
	if mlk.Receipt.ExitCode != 0 {
		panic(fmt.Sprintf("message exit %d", mlk.Receipt.ExitCode))
	}

	info, err := client.StateVoterInfo(ctx, from, types.EmptyTSK)
	if err != nil {
		panic(err)
	}
	totalVotes := big.Zero()
	for _, cand := range info.Candidates {
		totalVotes = big.Add(totalVotes, cand)
	}
	t.RecordMessage("rescind msg at %d, total %s, rewards %s, candidates: %v", mlk.Height, types.EPK(totalVotes), types.EPK(info.WithdrawableRewards), convertCandidateList(info.Candidates))

	return mlk.Height
}

func VoteWithdraw(t *TestEnvironment, ctx context.Context, client api.FullNode, from, to address.Address) abi.ChainEpoch {
	t.RecordMessage("withdraw all funds of %s to %s", from, to)

	var err error
	sp, err := actors.SerializeParams(&to)
	if err != nil {
		panic(err)
	}

	msg, err := client.MpoolPushMessage(ctx, &types.Message{
		To:     builtin2.VoteFundActorAddr,
		From:   from,
		Value:  big.Zero(),
		Method: builtin2.MethodsVote.Withdraw,
		Params: sp,
	}, nil)
	if err != nil {
		panic(err)
	}

	t.RecordMessage("send withdraw message: %s", msg.Cid())

	// wait
	mlk, err := client.StateWaitMsg(ctx, msg.Cid(), 1)
	if err != nil {
		panic(err)
	}
	if mlk.Receipt.ExitCode != 0 {
		panic(fmt.Sprintf("message exit %d", mlk.Receipt.ExitCode))
	}

	var out abi.TokenAmount
	err = out.UnmarshalCBOR(bytes.NewReader(mlk.Receipt.Return))
	if err != nil {
		panic(err)
	}

	t.RecordMessage("withdraw msg %s at %d to %s", types.EPK(out), mlk.Height, to)

	return mlk.Height
}

var initWithdrawtoBalance = abi.NewTokenAmount(1e18) // 1EPK
func InitWithdrawto(t *TestEnvironment, ctx context.Context, client api.FullNode, from, withdrawto address.Address) {
	msg, err := client.MpoolPushMessage(ctx, &types.Message{
		To:     withdrawto,
		From:   from,
		Value:  initWithdrawtoBalance,
		Method: builtin2.MethodSend,
	}, nil)
	if err != nil {
		panic(err)
	}

	mlk, err := client.StateWaitMsg(ctx, msg.Cid(), 1)
	if err != nil {
		panic(err)
	}
	if mlk.Receipt.ExitCode != 0 {
		panic(fmt.Sprintf("message exit %d", mlk.Receipt.ExitCode))
	}

	t.RecordMessage("init new withdrawTo %s at %d message: %s", withdrawto, mlk.Height, msg.Cid())
}

func RunCheckVoteRewards(t *TestEnvironment, ctx context.Context, client api.FullNode, voters []*CheckedClientsAddressesMsg, quit <-chan error) {
	height := 0
	headlag := 3

	cctx, cancel := context.WithCancel(ctx)
	defer cancel()

	tipsetsCh, err := tstats.GetTips(cctx, client, abi.ChainEpoch(height), headlag)
	if err != nil {
		panic(err)
	}

	idaVoter := make(map[address.Address]address.Address)
	var withdrawTos []address.Address
	for _, voter := range voters {
		ida, err := client.StateLookupID(ctx, voter.WalletAddr, types.EmptyTSK)
		if err != nil {
			panic(err)
		}
		idaVoter[ida] = voter.WalletAddr

		if voter.WithdrawTo != address.Undef {
			withdrawTos = append(withdrawTos, voter.WithdrawTo)
		}
	}

	// foundation is initial fallback
	genTs, err := client.ChainGetGenesis(ctx)
	if err != nil {
		panic(err)
	}
	genact, err := client.StateGetActor(ctx, builtin.FoundationIDAddress, genTs.Key())
	if err != nil {
		panic(err)
	}

	for tipset := range tipsetsCh {
		select {
		case err := <-quit:
			t.RecordMessage("@%d checker stopped: %w", tipset.Height(), err)
			return
		default:
		}

		tally, err := client.StateVoteTally(ctx, tipset.Key())
		if err != nil {
			panic(err)
		}

		if tally.FallbackReceiver != builtin.FoundationIDAddress {
			t.RecordFailure(xerrors.Errorf("fallback %s not equal foundation %s", tally.FallbackReceiver, builtin.FoundationIDAddress))
			return
		}

		act, err := client.StateGetActor(ctx, builtin.FoundationIDAddress, tipset.Key())
		if err != nil {
			panic(err)
		}
		foundation := big.Sub(act.Balance, genact.Balance)
		totalVoteRewards := big.Add(foundation, tally.UnownedFunds)
		t.RecordMessage("@%d sum rewards %s (foundation %s, unowned %s)", tipset.Height(), types.EPK(totalVoteRewards), types.EPK(foundation), types.EPK(tally.UnownedFunds))

		sumVotes := big.Zero()
		sumCandVotes := make(map[string]abi.TokenAmount)
		sumBalance := tally.UnownedFunds // unowned funds + voter balance[votes,rewards]
		for ida := range idaVoter {
			info, err := client.StateVoterInfo(ctx, ida, tipset.Key())
			if err != nil {
				if strings.Contains(err.Error(), "not found") {
					t.RecordMessage("@%d voter %s not found, skip", tipset.Height(), ida)
					continue
				}
				panic(err)
			}
			voterBalance := big.Sum(info.UnlockedVotes, info.UnlockingVotes, info.WithdrawableRewards)

			totalVoteRewards = big.Add(totalVoteRewards, info.WithdrawableRewards)

			for candStr, votes := range info.Candidates {
				sumVotes = big.Add(sumVotes, votes)
				voterBalance = big.Add(voterBalance, votes)

				old, ok := sumCandVotes[candStr]
				if !ok {
					sumCandVotes[candStr] = votes
					continue
				}
				sumCandVotes[candStr] = big.Add(old, votes)
			}

			t.RecordMessage("@%d voter %s: balance %s (unlocked %s, unlocking %s, withdrawable %s)", tipset.Height(), ida,
				types.EPK(voterBalance), types.EPK(info.UnlockedVotes), types.EPK(info.UnlockingVotes), types.EPK(info.WithdrawableRewards))

			sumBalance = big.Add(sumBalance, voterBalance)
		}

		sumWithdrawn := big.Zero()
		for _, wto := range withdrawTos {
			act, err := client.StateGetActor(ctx, wto, tipset.Key())
			if err != nil {
				if strings.Contains(err.Error(), "not found") {
					t.RecordMessage("@%d withdrawto %s not found, skip", tipset.Height(), wto)
					continue
				}
				panic(err)
			}
			ab := big.Sub(act.Balance, initWithdrawtoBalance)
			if ab.LessThan(big.Zero()) {
				panic("negative withdrawn amount")
			}
			sumWithdrawn = big.Add(sumWithdrawn, ab)
			t.RecordMessage("@%d withdrawto %s withdrawn %s", tipset.Height(), wto, types.EPK(ab))
		}

		// Expect: total votes == sum votes from all voters
		if !tally.TotalVotes.Equals(sumVotes) {
			t.RecordFailure(xerrors.Errorf("@%d total votes mismatched, %s - %s", tipset.Height(), types.EPK(tally.TotalVotes), types.EPK(sumVotes)))
			return
		}
		// Expect: candidate votes == sum votes for him from all voters
		for candStr, votes := range sumCandVotes {
			v, ok := tally.Candidates[candStr]
			if !ok {
				t.RecordFailure(xerrors.Errorf("@%d candidate %s not found in tally", tipset.Height(), candStr))
				return
			}
			if !v.Equals(votes) {
				t.RecordFailure(xerrors.Errorf("@%d candidate % votes mismatched, %s - %s", tipset.Height(), candStr, types.EPK(v), types.EPK(votes)))
				return
			}
		}

		t.RecordMessage("@%d total votes %s, unowned funds %s, candidates %s", tipset.Height(), types.EPK(tally.TotalVotes), types.EPK(tally.UnownedFunds), convertCandidateList(tally.Candidates))

		// Expect: mined vote rewards == voters' unwithdrawn + voters' withdrawn + fallback gained + unowned funds
		minedDetail, err := client.StateTotalMinedDetail(ctx, tipset.Key())
		if err != nil {
			panic(err)
		}

		t.RecordMessage("@%d mined reward %s: withdrawn %s, other %s", tipset.Height(), types.EPK(minedDetail.TotalVoteReward), types.EPK(sumWithdrawn), types.EPK(totalVoteRewards))
		if !minedDetail.TotalVoteReward.Equals(big.Add(totalVoteRewards, sumWithdrawn)) {
			t.RecordFailure(xerrors.Errorf("@%d mined rewards mismatched, expect %s", tipset.Height(), types.EPK(minedDetail.TotalVoteReward)))
			return
		}

		// Expect: vote fund balance == sum of voter's balance(votes+rewards) and unowned funds
		fundact, err := client.StateGetActor(ctx, builtin2.VoteFundActorAddr, tipset.Key())
		if err != nil {
			panic(err)
		}
		if !fundact.Balance.Equals(sumBalance) {
			t.RecordFailure(xerrors.Errorf("@%d vote fund balance mismatched: %s - %s", tipset.Height(), types.EPK(fundact.Balance), types.EPK(sumBalance)))
			return
		}

		t.RecordMessage("@%d voters check ok: mined vote rewards %s, fund balance %s (unowned %s)", tipset.Height(),
			types.EPK(minedDetail.TotalVoteReward), types.EPK(fundact.Balance), types.EPK(tally.UnownedFunds))
	}
}

func convertCandidateList(m map[string]big.Int) string {
	s := ""
	for cand, amt := range m {
		s += fmt.Sprintf("%s:%s, ", cand, types.EPK(amt))
	}
	return "{" + strings.TrimRight(s, ", ") + "}"
}
