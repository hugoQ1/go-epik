package main

import (
	"fmt"
	"strings"
	"time"

	lapi "github.com/EpiK-Protocol/go-epik/api"
	cliutil "github.com/EpiK-Protocol/go-epik/cli/util"
	"github.com/EpiK-Protocol/go-epik/cmd/epik-expressman/workspace"
	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/ipfs/go-cid"
	"github.com/urfave/cli/v2"
	"golang.org/x/xerrors"
)

var uploadCmd = &cli.Command{
	Name:      "upload",
	Usage:     "Upload files in batch",
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

		ws, err := workspace.LoadUploadWorkspace(cctx.Args().First())
		if err != nil {
			return err
		}

		if ws.Done() {
			fmt.Println("Upload already finished!")
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

		// find expert
		expert, err := findExpert(ctx, from, nodeApi)
		if err != nil {
			return err
		}

		//
		// upload file
		fmt.Printf("Start uploading files...\n  Directory:   %s\n  Total files: %d\n  As:          %s\n", ws.Dir(), ws.FileCount(), from)
		start := time.Now()
		{
			if err = ws.ForEach(func(df workspace.DataFile) error {
				base := df.Filename

				fmt.Printf("> Upload %s\n", base)

				// publish deal
				fmt.Printf(">> Publish deal, this may take several minutes\n")
				if df.OnchainID == "" {
					c, err := nodeApi.ClientImportAndDeal(ctx, &lapi.ImportAndDealParams{
						Ref: lapi.FileRef{
							Path:  ws.AbsPath(df),
							IsCAR: false,
						},
						Miner:  maddr,
						Expert: expert,
						From:   from,
					})
					if err != nil {
						return xerrors.Errorf("failed to import %s: %w", base, err)
					}
					ds, err := nodeApi.ClientDealPieceCID(ctx, c.Root)
					if err != nil {
						return err
					}
					err = ws.SetOnchainID(df.Filename, ds.PieceCID.String())
					if err != nil {
						return err
					}
					fmt.Printf(">> Published, onchain ID %s\n", ds.PieceCID)
				} else {
					fmt.Printf(">> Already published, skip, onchain ID %s\n", df.OnchainID)
				}
				return nil
			}); err != nil {
				return xerrors.Errorf("failed to iterate when sending: %w", err)
			}
		}

		fmt.Printf("Flush pending deals\n")
		time.Sleep(time.Minute)
		err = minerApi.MarketPublishPendingDeals(ctx)
		if err != nil {
			return xerrors.Errorf("failed to publish pending deals: %w", err)
		}

		// wait onchain
		fmt.Printf("Wait for consensus...\n")
		{
			done := make(map[string]bool) // true - success, false - failed
			fileSno := make(map[string]abi.SectorNumber)
			for {
				select {
				case <-ctx.Done():
					fmt.Println("> Interrupted for context canceled")
					return nil
				default:
				}

				fmt.Printf("> Check onchain status\n")
				if err = ws.ForEach(func(df workspace.DataFile) error {

					fmt.Printf(">> File %s, onchain ID %s\n", df.Filename, df.OnchainID)

					base := df.Filename
					if _, ok := done[base]; ok {
						return nil
					}

					piece, err := cid.Decode(df.OnchainID)
					if err != nil {
						return xerrors.Errorf("failed to decode onchain ID %s: %w", df.OnchainID, err)
					}

					sno, ok := fileSno[base]
					if !ok {
						fmt.Printf(">>> Query piece info\n")
						count := 0
						for {
							select {
							case <-ctx.Done():
								fmt.Printf(">>> Interrupted for context canceled\n")
								return xerrors.Errorf("Interrupted")
							default:
							}

							pi, err := minerApi.PiecesGetPieceInfo(ctx, piece)
							if err != nil {
								if strings.Contains(err.Error(), "datastore: key not found") {
									time.Sleep(30 * time.Second)
									count++
									if count >= 20 { // 10mins
										return xerrors.Errorf("sleep too long for querying piece info -- %s", base)
									}
									if count%2 == 0 {
										fmt.Printf(">>> Piece info not found, sleep 60s\n")
									}
									continue
								}
								return err
							}

							fileSno[base] = pi.Deals[0].SectorID
							sno = pi.Deals[0].SectorID
							fmt.Printf(">>> Piece info found, sector %d, wait for 1 more confirmation\n", sno)
							time.Sleep(30 * time.Second) // wait 2 confirmations
							break
						}
					}

					fmt.Printf(">>> Wait sector sealed\n")

					status, err := minerApi.SectorsStatus(ctx, sno, false)
					if err != nil {
						return xerrors.Errorf("failed to check sector status for %s: %w", df.OnchainID, err)
					}

					switch status.State {
					case "Proving":
						fmt.Printf(">>> Sector sealed\n")
						done[base] = true
					case "FailedUnrecoverable", "FaultedFinal", "Removed", "FaultReported":
						fmt.Printf(">>> Sector sealing failed\n")
						done[base] = false
					case "WaitDeals":
						err := minerApi.SectorStartSealing(ctx, sno)
						if err != nil {
							return xerrors.Errorf("failed to trigger sealing sector %d: %s, %w", sno, base, err)
						}
						fmt.Printf(">>> Sleep 10s and trigger sealing\n")
						time.Sleep(10 * time.Second)
					default:
						// sealing
					}
					return nil
				}); err != nil {
					return xerrors.Errorf("iterating failed: %w", err)
				}

				if len(done) == ws.FileCount() {
					succCnt := 0
					for _, succ := range done {
						if succ {
							succCnt++
						}
					}
					fmt.Printf("> File storage done, success %d, failed %d\n", succCnt, len(done)-succCnt)
					break
				}
				fmt.Printf("> File storage ongoing, sleep 2mins\n")
				time.Sleep(2 * time.Minute)
			}
		}
		fmt.Println("Finalize")
		err = ws.Finalize()
		if err != nil {
			return xerrors.Errorf("failed to finailize: %w", err)
		}
		fmt.Printf("All done! %fs\n", time.Since(start).Seconds())
		return nil
	},
}
