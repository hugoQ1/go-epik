package cli

import (
	"fmt"
	"math/rand"
	"strconv"

	"github.com/urfave/cli/v2"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/go-state-types/abi"

	"github.com/EpiK-Protocol/go-epik/chain/actors"
	"github.com/EpiK-Protocol/go-epik/chain/actors/builtin/retrieval"
	"github.com/EpiK-Protocol/go-epik/chain/types"
)

var stressCmd = &cli.Command{
	Name:      "stress",
	Usage:     "stress test for message pool ",
	ArgsUsage: "[count]",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "from",
			Usage: "optionally specify the account to send funds from",
		},
		&cli.StringFlag{
			Name:  "gas-premium",
			Usage: "specify gas price to use in AttoEPK",
			Value: "0",
		},
		&cli.StringFlag{
			Name:  "gas-feecap",
			Usage: "specify gas fee cap to use in AttoEPK",
			Value: "0",
		},
		&cli.Int64Flag{
			Name:  "gas-limit",
			Usage: "specify gas limit",
			Value: 0,
		},
		&cli.Int64Flag{
			Name:  "type",
			Usage: "specify message type, 0 normal, 1 retrieve pledge",
			Value: 0,
		},
	},
	Action: func(cctx *cli.Context) error {
		if cctx.Args().Len() != 2 {
			return ShowHelp(cctx, fmt.Errorf("'stress' expects one arguments, count and value"))
		}

		srv, err := GetFullNodeServices(cctx)
		if err != nil {
			return err
		}
		defer srv.Close() //nolint:errcheck

		ctx := ReqContext(cctx)
		var params []SendParams

		def, err := srv.API().WalletDefaultAddress(ctx)
		if err != nil {
			return err
		}

		list, err := srv.API().WalletList(ctx)
		if err != nil {
			return err
		}

		fmt.Fprintf(cctx.App.Writer, "load wallet list: %d\n", len(list))

		count, err := strconv.ParseUint(cctx.Args().Get(0), 10, 64)
		if err != nil {
			return err
		}

		val, err := types.ParseEPK(cctx.Args().Get(1))
		if err != nil {
			return ShowHelp(cctx, fmt.Errorf("failed to parse amount: %w", err))
		}

		mtype := uint64(0)
		if cctx.IsSet("type") {
			mtype = cctx.Uint64("type")
		}

		i := uint64(0)
		for i < count {
			from := list[rand.Intn(len(list))]
			for from == def {
				from = list[rand.Intn(len(list))]
			}
			to := list[rand.Intn(len(list))]
			for to == def || to == from {
				to = list[rand.Intn(len(list))]
			}
			if mtype == 1 {
				serializeParams, err := actors.SerializeParams(&retrieval.PledgeParams{
					Address: to,
				})
				if err != nil {
					return xerrors.Errorf("serializing params failed: %w", err)
				}
				param := SendParams{
					From:   from,
					To:     retrieval.Address,
					Val:    abi.TokenAmount(val),
					Method: retrieval.Methods.Pledge,
					Params: serializeParams,
				}
				params = append(params, param)
			} else {
				param := SendParams{
					From: from,
					To:   to,
					Val:  abi.TokenAmount(val),
				}
				params = append(params, param)
			}
			i++
		}

		fmt.Fprintf(cctx.App.Writer, "batch send msgs:\n")
		cids, err := srv.BatchSend(ctx, params)

		if err != nil {
			return xerrors.Errorf("executing send: %w", err)
		}
		for _, cid := range cids {
			fmt.Fprintf(cctx.App.Writer, "%s\n", cid)
		}
		return nil
	},
}
