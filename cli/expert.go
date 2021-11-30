package cli

import (
	"bytes"
	"context"
	"fmt"

	lapi "github.com/EpiK-Protocol/go-epik/api"
	"github.com/EpiK-Protocol/go-epik/build"
	"github.com/EpiK-Protocol/go-epik/chain/actors"
	types "github.com/EpiK-Protocol/go-epik/chain/types"
	"github.com/EpiK-Protocol/go-epik/lib/tablewriter"
	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/big"
	"github.com/filecoin-project/specs-actors/v2/actors/builtin"
	"github.com/filecoin-project/specs-actors/v2/actors/builtin/expert"
	"github.com/filecoin-project/specs-actors/v2/actors/builtin/expertfund"
	"github.com/filecoin-project/specs-actors/v2/actors/builtin/vote"
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
		expertInfoCmd,
		expertFileCmd,
		expertListCmd,
		expertNominateCmd,
		expertClaimCmd,
		expertVote,
		expertSetOwnerCmd,
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

	addr, err := applyForExpert(ctx, api, peerid, gasPrice, cctx)
	if err != nil {
		return xerrors.Errorf("applying for expert failed: %w", err)
	}
	log.Infof("Created new expert: %s", addr)

	return nil
}

func applyForExpert(ctx context.Context, api lapi.FullNode, peerid peer.ID, gasPrice types.BigInt, cctx *cli.Context) (address.Address, error) {
	log.Info("Applying for expert message")

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

	params, err := actors.SerializeParams(&expertfund.ApplyForExpertParams{
		Owner: owner,
	})
	if err != nil {
		return address.Undef, err
	}

	createMessage := &types.Message{
		To:    builtin.ExpertFundActorAddr,
		From:  owner,
		Value: expert.ExpertApplyCost,

		Method: builtin.MethodsExpertFunds.ApplyForExpert,
		Params: params,
	}

	signed, err := api.MpoolPushMessage(ctx, createMessage, nil)
	if err != nil {
		return address.Undef, err
	}

	log.Infof("Pushed ApplyForExpert %s to Mpool", signed.Cid())
	log.Infof("Waiting for confirmation")

	mw, err := api.StateWaitMsg(ctx, signed.Cid(), build.MessageConfidence)
	if err != nil {
		return address.Undef, err
	}

	if mw.Receipt.ExitCode != 0 {
		return address.Undef, xerrors.Errorf("create expert failed: exit code %d", mw.Receipt.ExitCode)
	}

	var retval expertfund.ApplyForExpertReturn
	if err := retval.UnmarshalCBOR(bytes.NewReader(mw.Receipt.Return)); err != nil {
		return address.Undef, err
	}

	log.Infof("New expert address is: %s (%s)", retval.IDAddress, retval.RobustAddress)
	return retval.IDAddress, nil
}

//expertInfoCmd
var expertInfoCmd = &cli.Command{
	Name:  "info",
	Usage: "expert info <expert>",
	Flags: []cli.Flag{},
	Action: func(cctx *cli.Context) error {
		ctx := ReqContext(cctx)

		if cctx.Args().Len() != 1 {
			return fmt.Errorf("usage: info <expert>")
		}

		expertAddr, err := address.NewFromString(cctx.Args().First())
		if err != nil {
			return err
		}

		api, closer, err := GetFullNodeAPI(cctx) // TODO: consider storing full node address in config
		if err != nil {
			return err
		}
		defer closer()

		info, err := api.StateExpertInfo(ctx, expertAddr, types.EmptyTSK)
		if err != nil {
			return err
		}
		fmt.Printf("Expert: %s\n", expertAddr)
		fmt.Printf("\tOwner: %s\n", info.Owner)
		fmt.Printf("\tProposer: %s\n", info.Proposer)
		expertType := "foundation"
		if info.Type == builtin.ExpertNormal {
			expertType = "normal"
		}
		fmt.Printf("\tType: %s\n", expertType)
		fmt.Printf("\tHash: %s\n", info.ApplicationHash)

		fmt.Printf("\nVotes: %s (required %s)\n", types.EPK(info.CurrentVotes), types.EPK(info.RequiredVotes))
		fmt.Printf("\tImplicated: %d times\n", info.ImplicatedTimes)
		fmt.Printf("\tData Count: %d\n", info.DataCount)
		fmt.Printf("\tStatus: %d (%s)\n", info.Status, info.StatusDesc)
		if info.Status == expert.ExpertStateUnqualified {
			head, err := api.ChainHead(ctx)
			if err != nil {
				return err
			}
			elapsed := head.Height() - info.LostEpoch
			fmt.Printf("Will lose shares in %d epochs (since epoch %d)\n", expertfund.ClearExpertContributionDelay-elapsed, info.LostEpoch)
		}

		fmt.Printf("\nLockRewards: %d\n", types.EPK(info.LockAmount))
		fmt.Printf("\nUnlockRewards: %d\n", types.EPK(info.UnlockAmount))
		fmt.Printf("\nTotalRewards: %d\n", types.EPK(info.TotalReward))
		return nil
	},
}

var expertFileCmd = &cli.Command{
	Name:  "file",
	Usage: "Interact with epik expert",
	Subcommands: []*cli.Command{
		fileRegisterCmd,
		fileListCmd,
	},
}

var fileRegisterCmd = &cli.Command{
	Name:      "register",
	Usage:     "expert register file",
	ArgsUsage: "[expert] [rootId]",
	Action: func(cctx *cli.Context) error {

		if cctx.Args().Len() != 2 {
			return fmt.Errorf("usage: register <expert> <rootId>")
		}

		expert, err := address.NewFromString(cctx.Args().First())
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

		root, err := cid.Parse(cctx.Args().Get(1))
		if err != nil {
			return err
		}

		ds, err := api.ClientDealPieceCID(ctx, root)
		if err != nil {
			return xerrors.Errorf("failed to get data cid/size for root %s: %w", root, err)
		}

		msg, err := api.ClientExpertRegisterFile(ctx, &lapi.ExpertRegisterFileParams{
			Expert:    expert,
			RootID:    root,
			PieceID:   ds.PieceCID,
			PieceSize: ds.PieceSize,
		})
		if err != nil {
			return xerrors.Errorf("failed to push register msg: %w", root, err)
		}
		fmt.Println("register CID: ", msg)
		return nil
	},
}

var fileListCmd = &cli.Command{
	Name:      "list",
	Usage:     "expert list file",
	ArgsUsage: "[expert]",
	Action: func(cctx *cli.Context) error {

		if cctx.Args().Len() != 1 {
			return fmt.Errorf("usage: list <expert>")
		}

		expert, err := address.NewFromString(cctx.Args().First())
		if err != nil {
			return err
		}

		ctx := ReqContext(cctx)

		api, closer, err := GetFullNodeAPI(cctx) // TODO: consider storing full node address in config
		if err != nil {
			return err
		}
		defer closer()

		infos, err := api.StateExpertDatas(ctx, expert, nil, false, types.EmptyTSK)

		w := tablewriter.New(
			tablewriter.Col("RootID"),
			tablewriter.Col("PieceCID"),
			tablewriter.Col("Size"))

		for _, d := range infos {
			if d.RootID == d.PieceID {
				// ignore fake data
				continue
			}
			w.Write(map[string]interface{}{
				"RootID":   d.RootID,
				"PieceCID": d.PieceID,
				"Size":     types.SizeStr(types.NewInt(uint64(d.PieceSize))),
			})
		}

		return w.Flush(cctx.App.Writer)
	},
}

var expertListCmd = &cli.Command{
	Name:  "list",
	Usage: "expert list",
	Flags: []cli.Flag{},
	Action: func(cctx *cli.Context) error {
		ctx := ReqContext(cctx)

		api, closer, err := GetFullNodeAPI(cctx)
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

var expertClaimCmd = &cli.Command{
	Name:      "claim",
	Usage:     "Claim expert rewards",
	ArgsUsage: "<expertAddress> <amount (EPK)>",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:    "from",
			Usage:   "optionally specify the owner account, otherwise it will use the default wallet address",
			Aliases: []string{"f"},
		},
	},
	Action: func(cctx *cli.Context) error {
		if cctx.Args().Len() != 2 {
			return ShowHelp(cctx, fmt.Errorf("'claim' expects two arguments, expert address and amount"))
		}

		api, closer, err := GetFullNodeAPI(cctx)
		if err != nil {
			return err
		}
		defer closer()

		ctx := ReqContext(cctx)

		expertAddr, err := address.NewFromString(cctx.Args().Get(0))
		if err != nil {
			return ShowHelp(cctx, fmt.Errorf("failed to parse expert address: %w", err))
		}

		amount, err := types.ParseEPK(cctx.Args().Get(1))
		if err != nil {
			return ShowHelp(cctx, fmt.Errorf("failed to parse amount: %w", err))
		}

		fromAddr, err := parseFrom(cctx, ctx, api, true)
		if err != nil {
			return err
		}

		sp, err := actors.SerializeParams(&expertfund.ClaimFundParams{
			Expert: expertAddr,
			Amount: big.Int(amount),
		})
		if err != nil {
			return xerrors.Errorf("serializing params: %w", err)
		}

		smsg, err := api.MpoolPushMessage(ctx, &types.Message{
			To:     builtin.ExpertFundActorAddr,
			From:   fromAddr,
			Value:  big.Zero(),
			Method: builtin.MethodsExpertFunds.Claim,
			Params: sp,
		}, nil)
		if err != nil {
			return xerrors.Errorf("failed to send claim message: %w", err)
		}

		fmt.Printf("Send claim message: %s\n", smsg.Cid())

		return nil
	},
}

var expertSetOwnerCmd = &cli.Command{
	Name:      "set-owner",
	Usage:     "Change owner address by old owner",
	ArgsUsage: "<expertAddress> <newOwnerAddress>",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:    "from",
			Usage:   "optionally specify the owner account, otherwise it will use the default wallet address",
			Aliases: []string{"f"},
		},
	},
	Action: func(cctx *cli.Context) error {
		if cctx.Args().Len() != 2 {
			return ShowHelp(cctx, fmt.Errorf("'set-owner' expects two arguments, expert address and new owner address"))
		}

		api, closer, err := GetFullNodeAPI(cctx)
		if err != nil {
			return err
		}
		defer closer()

		ctx := ReqContext(cctx)

		expertAddr, err := address.NewFromString(cctx.Args().Get(0))
		if err != nil {
			return ShowHelp(cctx, fmt.Errorf("failed to parse expert address: %w", err))
		}

		newOwnerAddr, err := address.NewFromString(cctx.Args().Get(1))
		if err != nil {
			return ShowHelp(cctx, fmt.Errorf("failed to parse new owner address: %w", err))
		}

		fromAddr, err := parseFrom(cctx, ctx, api, true)
		if err != nil {
			return err
		}

		sp, err := actors.SerializeParams(&newOwnerAddr)
		if err != nil {
			return xerrors.Errorf("serializing params: %w", err)
		}

		smsg, err := api.MpoolPushMessage(ctx, &types.Message{
			To:     expertAddr,
			From:   fromAddr,
			Value:  big.Zero(),
			Method: builtin.MethodsExpert.ChangeOwner,
			Params: sp,
		}, nil)
		if err != nil {
			return xerrors.Errorf("failed to send message: %w", err)
		}

		fmt.Printf("Send message: %s\n", smsg.Cid())

		mw, err := api.StateWaitMsg(ctx, smsg.Cid(), 2)
		if err != nil {
			return err
		}

		if mw.Receipt.ExitCode != 0 {
			return xerrors.Errorf("change expert owner failed: exit code %d", mw.Receipt.ExitCode)
		}

		fmt.Printf("Owner changed!\n")

		return nil
	},
}

var expertVote = &cli.Command{
	Name:  "votes",
	Usage: "Manage votes for experts",
	Subcommands: []*cli.Command{
		expertVoteSend,
		expertVoteRescind,
		expertVoteWithdraw,
		expertVoteInject,
	},
}

var expertVoteSend = &cli.Command{
	Name:      "send-votes",
	Usage:     "Send votes for an expert",
	ArgsUsage: "[expertAddress] [amount (EPK, one EPK one Vote)]",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:    "from",
			Usage:   "optionally specify the voter account, otherwise it will use the default wallet address",
			Aliases: []string{"f"},
		},
	},
	Action: func(cctx *cli.Context) error {
		if cctx.Args().Len() != 2 {
			return ShowHelp(cctx, fmt.Errorf("'vote' expects two arguments, expert and amount"))
		}

		api, closer, err := GetFullNodeAPI(cctx)
		if err != nil {
			return err
		}
		defer closer()

		ctx := ReqContext(cctx)

		expertAddr, err := address.NewFromString(cctx.Args().Get(0))
		if err != nil {
			return ShowHelp(cctx, fmt.Errorf("failed to parse expert address: %w", err))
		}

		val, err := types.ParseEPK(cctx.Args().Get(1))
		if err != nil {
			return ShowHelp(cctx, fmt.Errorf("failed to parse amount: %w", err))
		}

		fromAddr, err := parseFrom(cctx, ctx, api, true)
		if err != nil {
			return err
		}

		sp, err := actors.SerializeParams(&expertAddr)
		if err != nil {
			return xerrors.Errorf("serializing params: %w", err)
		}

		smsg, err := api.MpoolPushMessage(ctx, &types.Message{
			To:     builtin.VoteFundActorAddr,
			From:   fromAddr,
			Value:  types.BigInt(val),
			Method: builtin.MethodsVote.Vote,
			Params: sp,
		}, nil)
		if err != nil {
			return xerrors.Errorf("Submitting vote message: %w", err)
		}

		fmt.Printf("Vote message cid: %s\n", smsg.Cid())

		return nil
	},
}

var expertVoteRescind = &cli.Command{
	Name:      "rescind-votes",
	Usage:     "Rescind votes for an expert",
	ArgsUsage: "[expertAddress] [amount (EPK)]",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:    "from",
			Usage:   "optionally specify the voter account, otherwise it will use the default wallet address",
			Aliases: []string{"f"},
		},
	},
	Action: func(cctx *cli.Context) error {
		if cctx.Args().Len() != 2 {
			return ShowHelp(cctx, fmt.Errorf("'rescind' expects two arguments, expert and amount"))
		}

		api, closer, err := GetFullNodeAPI(cctx)
		if err != nil {
			return err
		}
		defer closer()

		ctx := ReqContext(cctx)

		expertAddr, err := address.NewFromString(cctx.Args().Get(0))
		if err != nil {
			return ShowHelp(cctx, fmt.Errorf("failed to parse expert address: %w", err))
		}

		val, err := types.ParseEPK(cctx.Args().Get(1))
		if err != nil {
			return ShowHelp(cctx, fmt.Errorf("failed to parse amount: %w", err))
		}

		fromAddr, err := parseFrom(cctx, ctx, api, true)
		if err != nil {
			return err
		}

		p := vote.RescindParams{
			Candidate: expertAddr,
			Votes:     types.BigInt(val),
		}
		sp, err := actors.SerializeParams(&p)
		if err != nil {
			return xerrors.Errorf("serializing params: %w", err)
		}

		smsg, err := api.MpoolPushMessage(ctx, &types.Message{
			To:     builtin.VoteFundActorAddr,
			From:   fromAddr,
			Value:  big.Zero(),
			Method: builtin.MethodsVote.Rescind,
			Params: sp,
		}, nil)
		if err != nil {
			return xerrors.Errorf("Submitting rescind message: %w", err)
		}

		fmt.Printf("Rescind message cid: %s\n", smsg.Cid())

		return nil
	},
}

var expertVoteWithdraw = &cli.Command{
	Name:  "withdraw",
	Usage: "Withdraw all unlocked votes and rewards",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:    "from",
			Usage:   "optionally specify the voter account, otherwise it will use the default wallet address",
			Aliases: []string{"f"},
		},
	},
	Action: func(cctx *cli.Context) error {

		api, closer, err := GetFullNodeAPI(cctx)
		if err != nil {
			return err
		}
		defer closer()

		ctx := ReqContext(cctx)

		fromAddr, err := parseFrom(cctx, ctx, api, true)
		if err != nil {
			return err
		}

		smsg, err := api.MpoolPushMessage(ctx, &types.Message{
			To:     builtin.VoteFundActorAddr,
			From:   fromAddr,
			Value:  big.Zero(),
			Method: builtin.MethodsVote.Withdraw,
			Params: nil,
		}, nil)
		if err != nil {
			return xerrors.Errorf("Submitting withdraw message: %w", err)
		}

		fmt.Printf("Withdraw message cid: %s\n", smsg.Cid())

		return nil
	},
}

var expertVoteInject = &cli.Command{
	Name:  "inject",
	Usage: "inject amount to vote pool",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:    "from",
			Usage:   "optionally specify the voter account, otherwise it will use the default wallet address",
			Aliases: []string{"f"},
		},
	},
	Action: func(cctx *cli.Context) error {
		if cctx.Args().Len() != 1 {
			return ShowHelp(cctx, fmt.Errorf("'inject' expects one argument, amount"))
		}

		api, closer, err := GetFullNodeAPI(cctx)
		if err != nil {
			return err
		}
		defer closer()

		ctx := ReqContext(cctx)

		amount, err := types.ParseEPK(cctx.Args().First())
		if err != nil {
			return ShowHelp(cctx, fmt.Errorf("failed to parse amount: %w", err))
		}

		fromAddr, err := parseFrom(cctx, ctx, api, true)
		if err != nil {
			return err
		}

		smsg, err := api.MpoolPushMessage(ctx, &types.Message{
			To:     builtin.VoteFundActorAddr,
			From:   fromAddr,
			Value:  big.Int(amount),
			Method: builtin.MethodsVote.InjectBalance,
			Params: nil,
		}, nil)
		if err != nil {
			return xerrors.Errorf("Submitting inject message: %w", err)
		}

		fmt.Printf("inject message cid: %s\n", smsg.Cid())

		return nil
	},
}
