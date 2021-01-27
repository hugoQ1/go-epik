package main

import (
	"context"
	"math/rand"
	"time"

	"github.com/EpiK-Protocol/go-epik/chain/types"
	"github.com/EpiK-Protocol/go-epik/chain/wallet"
	"github.com/EpiK-Protocol/go-epik/testplans/lotus-soup/testkit"
	"github.com/filecoin-project/go-address"
	"golang.org/x/xerrors"
)

func ecoVote(t *testkit.TestEnvironment) error {
	// Dispatch/forward non-client roles to defaults.
	if t.Role != "client" {
		return testkit.HandleDefaultRole(t)
	}

	cl, err := testkit.PrepareClient(t)
	if err != nil {
		return err
	}

	ctx := context.Background()
	client := cl.FullApi

	isChecker := t.BooleanParam("clients_checker")
	if isChecker {
		voters, err := testkit.CollectCheckedClientAddrs(t, ctx, t.IntParam("clients")-1)
		if err != nil {
			t.RecordMessage("checker failed to collect voter addresses: %w", err)
			return err
		}
		t.RecordMessage("start vote checker for: %v", voters)

		testkit.RunCheckVoteRewards(t, ctx, client, voters, t.SyncClient.MustBarrier(ctx, testkit.StateVoterDone, len(voters)).C)
		t.RecordMessage("checker finished")

	} else {
		var withdrawto address.Address
		from := cl.LotusNode.Wallet.Address

		doWith := t.BooleanParam("do_withdraw")
		if doWith {
			walletKey, err := wallet.GenerateKey(types.KTBLS)
			if err != nil {
				return err
			}
			withdrawto = walletKey.Address
			testkit.InitWithdrawto(t, ctx, client, from, withdrawto)
		}

		t.SyncClient.MustPublish(ctx, testkit.CheckedClientAddrsTopic, &testkit.CheckedClientsAddressesMsg{
			WalletAddr: from,
			WithdrawTo: withdrawto,
		})

		from, err = client.StateLookupID(ctx, from, types.EmptyTSK) // convert to id
		if err != nil {
			return err
		}

		// expert
		experts, err := client.StateListExperts(ctx, types.EmptyTSK)
		if err != nil {
			return err
		}
		if len(experts) == 0 {
			return xerrors.New("no experts")
		}
		expert := experts[0]

		// start ops
		epochs := t.IntParam("sleep_epochs")
		delay := rand.Intn(epochs) + 1
		t.RecordMessage("sleep %d epochs before send vote", delay)
		testkit.SleepEpochs(t, ctx, client, delay)

		// send
		height := testkit.VoteSend(t, ctx, client, from, expert)
		if height > 0 {
			t.RecordMessage("sleep %d epochs before withdraw-1 vote", epochs)
			testkit.SleepEpochs(t, ctx, client, epochs)
		}

		if doWith {
			// withdraw
			testkit.VoteWithdraw(t, ctx, client, from, withdrawto)
			t.RecordMessage("sleep %d epochs before rescind vote", epochs)
			testkit.SleepEpochs(t, ctx, client, epochs)
		}

		// rescind
		height = testkit.VoteRescind(t, ctx, client, from, expert)
		if height > 0 {
			t.RecordMessage("sleep %d epochs before withdraw-2 vote", epochs)
			testkit.SleepEpochs(t, ctx, client, epochs)
		}

		if doWith {
			// withdraw
			testkit.VoteWithdraw(t, ctx, client, from, withdrawto)
			testkit.SleepEpochs(t, ctx, client, epochs)
		}

		t.SyncClient.MustSignalEntry(ctx, testkit.StateVoterDone)
		t.RecordMessage("voter finished")
	}

	t.SyncClient.MustSignalEntry(ctx, testkit.StateStopMining)

	time.Sleep(10 * time.Second) // wait for metrics to be emitted

	t.SyncClient.MustSignalAndWait(ctx, testkit.StateDone, t.TestInstanceCount)
	return nil
}
