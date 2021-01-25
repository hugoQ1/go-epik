package cli

import (
	"bytes"
	"context"
	"fmt"

	lapi "github.com/EpiK-Protocol/go-epik/api"
	"github.com/EpiK-Protocol/go-epik/build"
	"github.com/EpiK-Protocol/go-epik/chain/actors"
	types "github.com/EpiK-Protocol/go-epik/chain/types"
	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/big"
	"github.com/filecoin-project/specs-actors/v2/actors/builtin"
	"github.com/filecoin-project/specs-actors/v2/actors/builtin/power"
	cid "github.com/ipfs/go-cid"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/urfave/cli/v2"
	"golang.org/x/xerrors"
)

var expertCmd = &cli.Command{
	Name:  "expert",
	Usage: "Manage expert and file",
	Subcommands: []*cli.Command{
		expertInitCmd,
		expertFileCmd,
		expertListCmd,
		expertNominateCmd,
	},
}

var expertInitCmd = &cli.Command{
	Name:  "init",
	Usage: "create expert",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:    "owner",
			Aliases: []string{"o"},
			Usage:   "owner key to use",
		},
		&cli.StringFlag{
			Name:  "gas-price",
			Usage: "set gas price for initialization messages in AttoEPK",
			Value: "0",
		},
		&cli.BoolFlag{
			Name:  "nosync",
			Usage: "don't check full-node sync status",
		},
	},
	Action: func(cctx *cli.Context) error {
		gasPrice, err := types.BigFromString(cctx.String("gas-price"))
		if err != nil {
			return xerrors.Errorf("failed to parse gas-price flag: %s", err)
		}

		ctx := ReqContext(cctx)

		log.Info("Trying to connect to full node RPC")

		api, closer, err := GetFullNodeAPI(cctx) // TODO: consider storing full node address in config
		if err != nil {
			return err
		}
		defer closer()

		log.Info("Checking full node sync status")

		if !cctx.Bool("nosync") {
			if err := SyncWait(ctx, api, false); err != nil {
				return xerrors.Errorf("sync wait: %w", err)
			}
		}

		if err := expertInit(ctx, cctx, api, gasPrice); err != nil {
			return xerrors.Errorf("Expert init failed: %w", err)
		}
		return nil
	},
}

func expertInit(ctx context.Context, cctx *cli.Context, api lapi.FullNode, gasPrice types.BigInt) error {
	peerid, err := api.ID(ctx)
	if err != nil {
		return xerrors.Errorf("peer ID from private key: %w", err)
	}

	addr, err := createExpert(ctx, api, peerid, gasPrice, cctx)
	if err != nil {
		return xerrors.Errorf("creating expert failed: %w", err)
	}
	log.Infof("Created new expert: %s", addr)

	return nil
}

func createExpert(ctx context.Context, api lapi.FullNode, peerid peer.ID, gasPrice types.BigInt, cctx *cli.Context) (address.Address, error) {
	log.Info("Creating expert message")

	var err error
	var owner address.Address
	if cctx.String("owner") != "" {
		owner, err = address.NewFromString(cctx.String("owner"))
	} else {
		owner, err = api.WalletDefaultAddress(ctx)
	}
	if err != nil {
		return address.Undef, err
	}

	params, err := actors.SerializeParams(&power.CreateExpertParams{
		Owner:  owner,
		PeerId: abi.PeerID(peerid),
	})
	if err != nil {
		return address.Undef, err
	}

	createMessage := &types.Message{
		To:    builtin.StoragePowerActorAddr,
		From:  owner,
		Value: big.Zero(),

		Method: builtin.MethodsPower.CreateExpert,
		Params: params,
	}

	signed, err := api.MpoolPushMessage(ctx, createMessage, nil)
	if err != nil {
		return address.Undef, err
	}

	log.Infof("Pushed power.CreateExpert, %s to Mpool", signed.Cid())
	log.Infof("Waiting for confirmation")

	mw, err := api.StateWaitMsg(ctx, signed.Cid(), build.MessageConfidence)
	if err != nil {
		return address.Undef, err
	}

	if mw.Receipt.ExitCode != 0 {
		return address.Undef, xerrors.Errorf("create expert failed: exit code %d", mw.Receipt.ExitCode)
	}

	var retval power.CreateExpertReturn
	if err := retval.UnmarshalCBOR(bytes.NewReader(mw.Receipt.Return)); err != nil {
		return address.Undef, err
	}

	log.Infof("New expert address is: %s (%s)", retval.IDAddress, retval.RobustAddress)
	return retval.IDAddress, nil
}

var expertFileCmd = &cli.Command{
	Name:  "file",
	Usage: "register file",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:    "expert",
			Aliases: []string{"e"},
			Usage:   "expert address",
		},
		&cli.StringFlag{
			Name:    "root",
			Aliases: []string{"r"},
			Usage:   "expert address",
		},
	},
	Action: func(cctx *cli.Context) error {

		expert, err := address.NewFromString(cctx.String("expert"))
		if err != nil {
			return err
		}

		ctx := ReqContext(cctx)

		// log.Info("Trying to connect to full node RPC")

		api, closer, err := GetFullNodeAPI(cctx) // TODO: consider storing full node address in config
		if err != nil {
			return err
		}
		defer closer()

		var root cid.Cid
		if cctx.String("root") != "" {
			parsed, err := cid.Parse(cctx.String("root"))
			if err != nil {
				return err
			}
			root = parsed
		}

		ds, err := api.ClientDealPieceCID(ctx, root)
		if err != nil {
			return xerrors.Errorf("failed to get data cid/size for root %s: %w", root, err)
		}

		msg, err := api.ClientExpertRegisterFile(ctx, &lapi.ExpertRegisterFileParams{
			Expert:    expert,
			PieceID:   ds.PieceCID,
			PieceSize: ds.PieceSize,
		})
		fmt.Println("register CID: ", msg)
		return nil
	},
}

var expertListCmd = &cli.Command{
	Name:  "list",
	Usage: "expert list",
	Flags: []cli.Flag{},
	Action: func(cctx *cli.Context) error {
		ctx := ReqContext(cctx)

		// log.Info("Trying to connect to full node RPC")

		api, closer, err := GetFullNodeAPI(cctx) // TODO: consider storing full node address in config
		if err != nil {
			return err
		}
		defer closer()

		list, err := api.StateListExperts(ctx, types.EmptyTSK)
		if err != nil {
			return err
		}
		for index, addr := range list {
			fmt.Printf("expert %d: %s\n", index, addr)
		}
		return nil
	},
}

//expertNominateCmd
var expertNominateCmd = &cli.Command{
	Name:  "nominate",
	Usage: "expert nominate <expert> <target>",
	Flags: []cli.Flag{},
	Action: func(cctx *cli.Context) error {
		ctx := ReqContext(cctx)

		if cctx.Args().Len() != 2 {
			return fmt.Errorf("usage: nominate <expert> <target>")
		}

		expert, err := address.NewFromString(cctx.Args().First())
		if err != nil {
			return err
		}

		target, err := address.NewFromString(cctx.Args().Get(1))
		if err != nil {
			return err
		}

		api, closer, err := GetFullNodeAPI(cctx) // TODO: consider storing full node address in config
		if err != nil {
			return err
		}
		defer closer()

		msg, err := api.ClientExpertNominate(ctx, expert, target)
		if err != nil {
			return err
		}
		fmt.Printf("expert nominate: %s\n", msg)
		return nil
	},
}
