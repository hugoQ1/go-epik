package test

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/EpiK-Protocol/go-epik/api"

	"github.com/stretchr/testify/require"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/sector-storage/mock"
	"github.com/filecoin-project/specs-actors/actors/abi"
	miner2 "github.com/filecoin-project/specs-actors/actors/builtin/miner"
	sealing "github.com/filecoin-project/storage-fsm"

	"github.com/EpiK-Protocol/go-epik/chain/types"
	"github.com/EpiK-Protocol/go-epik/node/impl"
)

func TestPledgeSector(t *testing.T, b APIBuilder, blocktime time.Duration, nSectors int) {
	os.Setenv("BELLMAN_NO_GPU", "1")

	ctx := context.Background()
	n, sn := b(t, 1, oneMiner)
	client := n[0].FullNode.(*impl.FullNodeAPI)
	miner := sn[0]

	addrinfo, err := client.NetAddrsListen(ctx)
	if err != nil {
		t.Fatal(err)
	}

	if err := miner.NetConnect(ctx, addrinfo); err != nil {
		t.Fatal(err)
	}
	time.Sleep(time.Second)

	mine := true
	done := make(chan struct{})
	go func() {
		defer close(done)
		for mine {
			time.Sleep(blocktime)
			if err := sn[0].MineOne(ctx, func(bool, error) {}); err != nil {
				t.Error(err)
			}
		}
	}()

	pledgeSectors(t, ctx, miner, nSectors)

	mine = false
	<-done
}

func pledgeSectors(t *testing.T, ctx context.Context, miner TestStorageNode, n int) {
	for i := 0; i < n; i++ {
		err := miner.PledgeSector(ctx)
		require.NoError(t, err)
	}

	for {
		s, err := miner.SectorsList(ctx) // Note - the test builder doesn't import genesis sectors into FSM
		require.NoError(t, err)
		fmt.Printf("Sectors: %d\n", len(s))
		if len(s) >= n {
			break
		}

		time.Sleep(100 * time.Millisecond)
	}

	fmt.Printf("All sectors is fsm\n")

	s, err := miner.SectorsList(ctx)
	require.NoError(t, err)

	toCheck := map[abi.SectorNumber]struct{}{}
	for _, number := range s {
		toCheck[number] = struct{}{}
	}

	for len(toCheck) > 0 {
		for n := range toCheck {
			st, err := miner.SectorsStatus(ctx, n)
			require.NoError(t, err)
			if st.State == api.SectorState(sealing.Proving) {
				delete(toCheck, n)
			}
			if strings.Contains(string(st.State), "Fail") {
				t.Fatal("sector in a failed state", st.State)
			}
		}

		time.Sleep(100 * time.Millisecond)
		fmt.Printf("WaitSeal: %d\n", len(s))
	}
}

func TestWindowPost(t *testing.T, b APIBuilder, blocktime time.Duration, nSectors int) {
	os.Setenv("BELLMAN_NO_GPU", "1")

	ctx := context.Background()
	n, sn := b(t, 1, oneMiner)
	client := n[0].FullNode.(*impl.FullNodeAPI)
	miner := sn[0]

	addrinfo, err := client.NetAddrsListen(ctx)
	if err != nil {
		t.Fatal(err)
	}

	if err := miner.NetConnect(ctx, addrinfo); err != nil {
		t.Fatal(err)
	}
	time.Sleep(time.Second)

	mine := true
	done := make(chan struct{})
	go func() {
		defer close(done)
		for mine {
			time.Sleep(blocktime)
			if err := sn[0].MineOne(ctx, func(bool, error) {}); err != nil {
				t.Error(err)
			}
		}
	}()

	maddr, err := miner.ActorAddress(ctx)
	require.NoError(t, err)

	mid, err := address.IDFromAddress(maddr)
	require.NoError(t, err)

	ssz, err := miner.ActorSectorSize(ctx, maddr)
	require.NoError(t, err)

	pledgeSectors(t, ctx, miner, nSectors)

	p, err := client.StateMinerPower(ctx, maddr, types.EmptyTSK)
	require.NoError(t, err)
	fmt.Println("power@init", p)
	require.Equal(t, p.MinerPower, p.TotalPower)
	require.Equal(t, p.MinerPower.RawBytePower, types.NewInt(uint64(ssz)*uint64(nSectors+GenesisPreseals)))

	di, err := client.StateMinerProvingDeadline(ctx, maddr, types.EmptyTSK)
	require.NoError(t, err)

	fmt.Printf("Running one proving periods\n")

	makeFailed := false
	mocksm := miner.StorageMiner.(*impl.StorageMinerAPI).Miner.ISectorManager().(*mock.SectorMgr)
	for {
		head, err := client.ChainHead(ctx)
		require.NoError(t, err)

		if head.Height() > di.PeriodStart+(miner2.WPoStProvingPeriod)+2 {
			break
		}

		// Drop sector 3&4 from deadline 1 partition 0
		if !makeFailed && head.Height() > di.PeriodStart+miner2.WPoStChallengeWindow+2 {
			makeFailed = true
			err = mocksm.MarkFailed(abi.SectorID{
				Miner:  abi.ActorID(mid),
				Number: abi.SectorNumber(3),
			}, true)
			require.NoError(t, err)
			err = mocksm.MarkFailed(abi.SectorID{
				Miner:  abi.ActorID(mid),
				Number: abi.SectorNumber(4),
			}, true)
			require.NoError(t, err)

			for {
				head, err := client.ChainHead(ctx)
				require.NoError(t, err)
				if head.Height() > di.PeriodStart+miner2.WPoStChallengeWindow*2+2 {
					break
				}
				if head.Height()%100 == 0 {
					fmt.Printf("After dropping sector 3 @%d\n", head.Height())
				}
				time.Sleep(blocktime)
			}
		}

		if head.Height()%1000 == 0 {
			fmt.Printf("@%d\n", head.Height())
		}
		time.Sleep(blocktime)
	}

	p, err = client.StateMinerPower(ctx, maddr, types.EmptyTSK)
	require.NoError(t, err)
	fmt.Println("power@drop(sector[3 4])", p)
	require.Equal(t, p.MinerPower, p.TotalPower)
	require.Equal(t, types.NewInt(uint64(ssz)*uint64(nSectors+GenesisPreseals-2)), p.MinerPower.RawBytePower)

	// Recover sector 4
	err = mocksm.MarkFailed(abi.SectorID{
		Miner:  abi.ActorID(mid),
		Number: abi.SectorNumber(4),
	}, false)
	require.NoError(t, err)

	for {
		head, err := client.ChainHead(ctx)
		require.NoError(t, err)
		if head.Height() > di.PeriodStart+miner2.WPoStProvingPeriod+miner2.WPoStChallengeWindow*2+2 {
			break
		}
		if head.Height()%100 == 0 {
			fmt.Printf("After recovering sector 4 @%d\n", head.Height())
		}
		time.Sleep(blocktime)
	}

	p, err = client.StateMinerPower(ctx, maddr, types.EmptyTSK)
	require.NoError(t, err)
	fmt.Println("power@recover4", p)
	require.Equal(t, p.MinerPower, p.TotalPower)
	require.Equal(t, types.NewInt(uint64(ssz)*uint64(nSectors+GenesisPreseals-1)), p.MinerPower.RawBytePower)

	mine = false
	<-done
}

func TestSectorsDist(t *testing.T, b APIBuilder, blocktime time.Duration, nSectors int) {
	// os.Setenv("BELLMAN_NO_GPU", "1")

	ctx := context.Background()
	n, sn := b(t, 1, oneMiner)
	client := n[0].FullNode.(*impl.FullNodeAPI)
	miner := sn[0]

	addrinfo, err := client.NetAddrsListen(ctx)
	if err != nil {
		t.Fatal(err)
	}

	if err := miner.NetConnect(ctx, addrinfo); err != nil {
		t.Fatal(err)
	}
	time.Sleep(time.Second)

	mine := true
	done := make(chan struct{})
	go func() {
		defer close(done)
		for mine {
			time.Sleep(blocktime)
			if err := sn[0].MineOne(ctx, func(bool, error) {}); err != nil {
				t.Error(err)
			}
		}
	}()

	maddr, err := miner.ActorAddress(ctx)
	require.NoError(t, err)

	pledgeSectors(t, ctx, miner, nSectors)

	di, err := client.StateMinerProvingDeadline(ctx, maddr, types.EmptyTSK)
	require.NoError(t, err)

	for {
		head, err := client.ChainHead(ctx)
		require.NoError(t, err)

		if head.Height() >= di.PeriodStart {

			fmt.Printf("First proving start %d, current head %d\n", di.PeriodStart, head.Height())

			deadlines, err := client.StateMinerDeadlines(ctx, maddr, types.EmptyTSK)
			require.NoError(t, err)

			nonZeroDeadline := 0
			sectors := make(map[uint64]int)
			for index, due := range deadlines.Due {
				cnt, err := due.Count()
				require.NoError(t, err)
				if cnt == 0 {
					continue
				}
				require.Greater(t, index, 0) // first deadline is left empty
				nonZeroDeadline++

				ds, err := due.All(uint64(nSectors+GenesisPreseals) * 2)
				require.NoError(t, err)
				fmt.Printf("deadline %d, sectors %v\n", index, ds)
				for _, sid := range ds {
					if prev, ok := sectors[sid]; ok {
						t.Fatalf("duplicated sector %d (prev deadline %d, current %d)", sid, prev, index)
					}
					sectors[sid] = index
				}
			}
			require.Equal(t, len(sectors), nSectors+GenesisPreseals)
			require.GreaterOrEqual(t, nonZeroDeadline, (nSectors+GenesisPreseals)/2) //
			break
		}

		if head.Height()%300 == 0 {
			fmt.Printf("@%d\n", head.Height())
		}
		time.Sleep(blocktime)
	}

	mine = false
	<-done
}
