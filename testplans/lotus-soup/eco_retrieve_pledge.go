package main

import (
	"context"
	"time"

	"github.com/EpiK-Protocol/go-epik/testplans/lotus-soup/testkit"
	"github.com/filecoin-project/go-address"
)

func ecoRetrievePledge(t *testkit.TestEnvironment) error {
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

		clients, err := testkit.CollectCheckedClientAddrs(t, ctx, t.IntParam("clients")-1)
		if err != nil {
			t.RecordMessage("checker failed to collect voter addresses: %w", err)
			return err
		}

		var genesisAccounts []address.Address
		for _, c := range clients {
			genesisAccounts = append(genesisAccounts, c.WalletAddr)
		}
		for _, m := range cl.MinerAddrs {
			genesisAccounts = append(genesisAccounts, m.WalletAddr)
		}

		t.RecordMessage("start rewards checker")
		testkit.RunCheckRewards(t, ctx, genesisAccounts, client, t.SyncClient.MustBarrier(ctx, testkit.StatePledgerDone, t.IntParam("clients")-1).C)
		t.RecordMessage("checker finished")

	} else {
		t.SyncClient.MustPublish(ctx, testkit.CheckedClientAddrsTopic, &testkit.CheckedClientsAddressesMsg{
			WalletAddr: cl.LotusNode.Wallet.Address,
		})

		epochs := t.IntParam("sleep_epochs")

		t.RecordMessage("sleep %d epochs before send pledge", epochs)
		testkit.SleepEpochs(t, ctx, client, epochs)

		height := testkit.SendRetrievePledge(t, ctx, client)
		if height > 0 {
			t.RecordMessage("sleep %d epochs after pledge at %d", epochs, height)
			testkit.SleepEpochs(t, ctx, client, epochs)
		}

		// FIXME: Due to limitations, WithdrawRetrievePledge cannot be covered
		// height = testkit.WithdrawRetrievePledge(t, ctx, client)
		// if height > 0 {
		// 	t.RecordMessage("sleep %d epochs after withdraw at %d", epochs, height)
		// 	testkit.SleepEpochs(t, ctx, client, epochs)
		// }

		t.SyncClient.MustSignalEntry(ctx, testkit.StatePledgerDone)

		t.RecordMessage("pledger finished")
	}

	t.SyncClient.MustSignalEntry(ctx, testkit.StateStopMining)

	time.Sleep(10 * time.Second) // wait for metrics to be emitted

	t.SyncClient.MustSignalAndWait(ctx, testkit.StateDone, t.TestInstanceCount)
	return nil
}
