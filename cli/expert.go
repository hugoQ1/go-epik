package cli

import (
	"bytes"
	"context"

	lapi "github.com/EpiK-Protocol/go-epik/api"
	"github.com/EpiK-Protocol/go-epik/build"
	"github.com/EpiK-Protocol/go-epik/chain/actors"
	types "github.com/EpiK-Protocol/go-epik/chain/types"
	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/big"
	"github.com/filecoin-project/specs-actors/actors/builtin"
	"github.com/filecoin-project/specs-actors/actors/builtin/power"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/urfave/cli/v2"
	"golang.org/x/xerrors"
)

var expertCmd = &cli.Command{
	Name:  "expert",
	Usage: "Manage expert",
	Subcommands: []*cli.Command{
		expertInitCmd,
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
			if err := SyncWait(ctx, api); err != nil {
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

		GasLimit: 10000000,
		GasPrice: gasPrice,
	}

	signed, err := api.MpoolPushMessage(ctx, createMessage)
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
