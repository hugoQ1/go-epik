package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	lapi "github.com/EpiK-Protocol/go-epik/api"
	"github.com/EpiK-Protocol/go-epik/build"
	"github.com/EpiK-Protocol/go-epik/chain/types"
	cliutil "github.com/EpiK-Protocol/go-epik/cli/util"
	"github.com/EpiK-Protocol/go-epik/cmd/epik-expressman/workspace"
	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-fil-markets/retrievalmarket"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/big"
	"github.com/ipfs/go-cid"
	"github.com/urfave/cli/v2"
	"golang.org/x/xerrors"
)

var downloadCmd = &cli.Command{
	Name:      "download",
	Usage:     "Download files in batch",
	ArgsUsage: "<path/to/directory>",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "from",
			Usage: "optional, specify expert owner address",
		},
	},
	Action: func(cctx *cli.Context) error {

		minerApi, closer, err := cliutil.GetStorageMinerAPI(cctx)
		if err != nil {
			return err
		}
		defer closer()

		nodeApi, closer, err := cliutil.GetFullNodeAPI(cctx)
		if err != nil {
			return err
		}
		defer closer()

		ctx := cliutil.ReqContext(cctx)

		maddr, err := minerApi.ActorAddress(ctx)
		if err != nil {
			return err
		}

		if !cctx.Args().Present() {
			return xerrors.New("'upload' requires one argument, directory path to be uploaded.")
		}

		ws, err := workspace.LoadDownloadWorkspace(cctx.Args().First())
		if err != nil {
			return err
		}

		if ws.Done() {
			fmt.Println("Download already finished!")
			return nil
		}

		// from
		var from address.Address
		if v := cctx.String("from"); v != "" {
			from, err = address.NewFromString(v)
			if err != nil {
				return xerrors.Errorf("failed to parse 'from' address: %w", err)
			}
		} else {
			from, err = nodeApi.WalletDefaultAddress(ctx)
			if err != nil {
				return err
			}
		}

		// download file
		fmt.Printf("Start downloading files...\n  Directory:   %s\n  Total files: %d\n  As:          %s\n", ws.Dir(), ws.FileCount(), from)
		start := time.Now()
		{
			if err = ws.ForEach(func(df workspace.DataFile) error {
				base := df.Filename
				absPath := ws.AbsPath(df)
				// file not uploaded
				if df.OnchainID == "" {
					return xerrors.Errorf("Onchain ID not set for file %s", base)
				}

				fmt.Printf("Download %s\n", base)

				exist, err := workspace.FileExists(absPath)
				if err != nil {
					return err
				}
				if exist {
					fmt.Printf("> Already downloaded, skip\n")
					return nil
				}

				tmp := absPath + ".tmp"
				if exist, err := workspace.FileExists(tmp); err != nil {
					return err
				} else if exist {
					if err = os.Remove(tmp); err != nil {
						return xerrors.Errorf("failed to delete temporary file %s", tmp)
					} else {
						fmt.Printf("> Remove cache file\n")
					}
				}

				// query deal id
				piece, err := cid.Decode(df.OnchainID)
				if err != nil {
					return xerrors.Errorf("failed decode onchain ID: %s, %s", df.OnchainID, df.Filename)
				}
				fmt.Printf("> Query deal...\n")
				count := 0
				var dealID abi.DealID
				for {
					select {
					case <-ctx.Done():
						fmt.Println(">> Interrupted for context canceled")
						return xerrors.Errorf("Interrupted")
					default:
					}
					pi, err := minerApi.PiecesGetPieceInfo(ctx, piece)
					if err != nil {
						if strings.Contains(err.Error(), "datastore: key not found") {
							time.Sleep(30 * time.Second)
							count++
							if count >= 20 { // 10mins
								return xerrors.Errorf("wait too long for deal ID query -- %s", base)
							}
							if count%2 == 0 {
								fmt.Printf(">> Wait deal sync...60s\n")
							}
							continue
						}
						return err
					}
					dealID = pi.Deals[0].DealID
					fmt.Printf(">> Deal ID is %d\n", dealID)
					break
				}

				// query root ID
				fmt.Printf("> Query root ID...\n")
				var rootID cid.Cid
				{
					deal, err := nodeApi.StateMarketStorageDeal(ctx, dealID, types.EmptyTSK)
					if err != nil {
						return xerrors.Errorf("failed to get storage deal %d of %s: %w", dealID, base, err)
					}
				outer:
					for i := deal.State.SectorStartEpoch; i < deal.State.SectorStartEpoch+abi.ChainEpoch(30); i++ {
						select {
						case <-ctx.Done():
							fmt.Println(">> Interrupted for context canceled")
							return xerrors.Errorf("Interrupted")
						default:
							indexes, err := nodeApi.StateDataIndex(ctx, i, types.EmptyTSK)
							if err != nil {
								return xerrors.Errorf("failed to query data index for %s at %d: %w", base, i, err)
							}
							for _, index := range indexes {
								if index.PieceCID == piece {
									if index.Miner == maddr {
										rootID = index.RootCID
										fmt.Printf(">> Root ID is %s, found at epoch %d\n", rootID, i)
										break outer
									}
								}
							}
						}
					}
				}
				if rootID == cid.Undef {
					return xerrors.Errorf("Root ID not found of file %s", base)
				}

				// check retrieval pledge
				fmt.Printf("> Check retrieval pledge\n")
				{
					state, err := nodeApi.StateRetrievalPledge(ctx, from, types.EmptyTSK)
					if err != nil {
						return xerrors.Errorf("error check retrieval pledge: %w", err)
					}
					avail := big.Sub(state.Balance, state.DayExpend)
					required := big.Div(big.Mul(big.NewIntUnsigned(build.EpkPrecision), big.NewIntUnsigned(uint64(8<<20))), big.NewInt(10<<20))
					fmt.Printf(">> Available %s, required %s\n", types.EPK(avail), types.EPK(required))
					if avail.LessThan(required) {
						amt := big.Sub(required, avail)
						fmt.Printf(">> Not enough, try adding pledge amount %s\n", types.EPK(amt))
						msgCid, err := nodeApi.ClientRetrievePledge(ctx, from, amt)
						if err != nil {
							return xerrors.Errorf("error adding retrieval pledge %s: %w", types.EPK(amt), err)
						}
						wait, err := nodeApi.StateWaitMsg(ctx, msgCid, 2)
						if err != nil {
							return xerrors.Errorf("error waiting msg %s: %w", msgCid, err)
						}
						if wait.Receipt.ExitCode != 0 {
							return fmt.Errorf("msg returned exit %d", wait.Receipt.ExitCode)
						}
						fmt.Printf(">> Pledge added\n")
					}
				}

				// save file
				{
					offer, err := nodeApi.ClientMinerQueryOffer(ctx, maddr, rootID, &piece)
					if err != nil {
						return xerrors.Errorf("error getting miner offer for %s: %w", base, err)
					}

					ref := &lapi.FileRef{
						Path:  tmp,
						IsCAR: false,
					}
					updates, err := nodeApi.ClientRetrieveWithEvents(ctx, offer.Order(from), ref)
					if err != nil {
						return xerrors.Errorf("error setting up retrieval: %w", err)
					}

					fmt.Printf("> Receive file from %s, saved to %s\n", maddr, absPath)

					for {
						select {
						case evt, ok := <-updates:
							if ok {
								fmt.Printf(">> Recv: %s, %s (%s)\n",
									types.SizeStr(types.NewInt(evt.BytesReceived)),
									retrievalmarket.ClientEvents[evt.Event],
									retrievalmarket.DealStatuses[evt.Status],
								)
							} else {
								fmt.Println(">> Success")
								return os.Rename(tmp, absPath)
							}

							if evt.Err != "" {
								return xerrors.Errorf("retrieval failed for %s: %s", base, evt.Err)
							}
						case <-ctx.Done():
							return xerrors.Errorf("retrieval timed out")
						}
					}
				}
			}); err != nil {
				return xerrors.Errorf("iterating failed: %w", err)
			}
		}

		fmt.Println("Finalize")
		err = ws.Finalize()
		if err != nil {
			return xerrors.Errorf("finailize error: %w", err)
		}
		fmt.Printf("All done! %fs\n", time.Since(start).Seconds())
		return nil
	},
}
