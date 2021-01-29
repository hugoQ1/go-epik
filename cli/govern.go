package cli

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/EpiK-Protocol/go-epik/api"
	"github.com/EpiK-Protocol/go-epik/blockstore"
	"github.com/EpiK-Protocol/go-epik/build"
	"github.com/EpiK-Protocol/go-epik/chain/actors"
	"github.com/EpiK-Protocol/go-epik/chain/actors/adt"
	"github.com/EpiK-Protocol/go-epik/chain/actors/builtin/multisig"
	"github.com/EpiK-Protocol/go-epik/chain/stmgr"
	types "github.com/EpiK-Protocol/go-epik/chain/types"
	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/big"
	"github.com/ipfs/go-cid"
	cbor "github.com/ipfs/go-ipld-cbor"
	"github.com/urfave/cli/v2"
	cbg "github.com/whyrusleeping/cbor-gen"
	"golang.org/x/xerrors"

	builtin2 "github.com/filecoin-project/specs-actors/v2/actors/builtin"
	"github.com/filecoin-project/specs-actors/v2/actors/builtin/expertfund"
	"github.com/filecoin-project/specs-actors/v2/actors/builtin/exported"
	gov2 "github.com/filecoin-project/specs-actors/v2/actors/builtin/govern"
	market2 "github.com/filecoin-project/specs-actors/v2/actors/builtin/market"
	msig2 "github.com/filecoin-project/specs-actors/v2/actors/builtin/multisig"
)

var govCmd = &cli.Command{
	Name:  "gov",
	Usage: "Govern EpiK network",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:    "from",
			Usage:   "Optionally specify the governor (or multisig signer) address, otherwise it will use the default wallet address",
			Aliases: []string{"f"},
		},
		&cli.StringFlag{
			Name:    "msig-governor",
			Usage:   "Optionally specify the multisig governor address",
			Aliases: []string{"m"},
		},
		&cli.IntFlag{
			Name:    "confidence",
			Usage:   "number of block confirmations to wait for",
			Value:   int(build.MessageConfidence),
			Aliases: []string{"c"},
		},
	},
	Subcommands: []*cli.Command{
		govAuth,
		govMarket,
		govKnowledge,
		govExpert,
		govExpertfund,
		govApprove,
	}}

//////////////////////////
//     gov auth
//////////////////////////

var govAuth = &cli.Command{
	Name:  "auth",
	Usage: "Manipulate governance authorizations",
	Subcommands: []*cli.Command{
		govAuthGrant,
		govAuthRevoke,
		govAuthListCmd,
	},
}

var govAuthGrant = &cli.Command{
	Name:      "grant",
	Usage:     "Propose granting priviledges to governor",
	ArgsUsage: "[governorAddress]",
	Action: func(cctx *cli.Context) error {
		if !cctx.Args().Present() {
			return ShowHelp(cctx, fmt.Errorf("'grant' expects one argument, governor address"))
		}

		api, closer, err := GetFullNodeAPI(cctx)
		if err != nil {
			return err
		}
		defer closer()

		ctx := ReqContext(cctx)

		// governor
		governor, err := address.NewFromString(cctx.Args().First())
		if err != nil {
			return fmt.Errorf("failed to parse governor address: %w", err)
		}
		ida, err := api.StateLookupID(ctx, governor, types.EmptyTSK)
		if err != nil {
			return fmt.Errorf("failed to lookup id address for %s", governor)
		}

		// from
		fromAddr, err := parseFrom(cctx, ctx, api, true)
		if err != nil {
			return err
		}

		// method param
		param := gov2.GrantOrRevokeParams{Governor: ida}
		err = PromptForGrantOrRevoke(&param)
		if err != nil {
			return fmt.Errorf("failed to read propose param: %w", err)
		}

		super, err := api.StateGovernSupervisor(ctx, types.EmptyTSK)
		if err != nil {
			return fmt.Errorf("failed to get govern supervisor: %w", err)
		}

		sp, err := actors.SerializeParams(&param)
		if err != nil {
			return fmt.Errorf("failed to serialize param: %w", err)
		}
		return sendProposal(cctx, ctx, api, super, builtin2.GovernActorAddr, fromAddr, big.Zero(), builtin2.MethodsGovern.Grant, sp)
	},
}

var govAuthRevoke = &cli.Command{
	Name:      "revoke",
	Usage:     "Propose revoking priviledges from governor",
	ArgsUsage: "[governorAddress]",
	Action: func(cctx *cli.Context) error {
		if !cctx.Args().Present() {
			return ShowHelp(cctx, fmt.Errorf("'revoke' expects one argument, governor address"))
		}

		api, closer, err := GetFullNodeAPI(cctx)
		if err != nil {
			return err
		}
		defer closer()

		ctx := ReqContext(cctx)

		// governor
		governor, err := address.NewFromString(cctx.Args().First())
		if err != nil {
			return fmt.Errorf("failed to parse governor address: %w", err)
		}
		ida, err := api.StateLookupID(ctx, governor, types.EmptyTSK)
		if err != nil {
			return fmt.Errorf("failed to lookup id address for %s", governor)
		}

		// from
		fromAddr, err := parseFrom(cctx, ctx, api, true)
		if err != nil {
			return err
		}

		// method param
		param := gov2.GrantOrRevokeParams{Governor: ida}
		err = PromptForGrantOrRevoke(&param)
		if err != nil {
			return fmt.Errorf("failed to read propose param: %w", err)
		}

		super, err := api.StateGovernSupervisor(ctx, types.EmptyTSK)
		if err != nil {
			return fmt.Errorf("failed to get govern supervisor: %w", err)
		}

		sp, err := actors.SerializeParams(&param)
		if err != nil {
			return fmt.Errorf("failed to serialize param: %w", err)
		}
		return sendProposal(cctx, ctx, api, super, builtin2.GovernActorAddr, fromAddr, big.Zero(), builtin2.MethodsGovern.Revoke, sp)
	},
}

var govAuthListCmd = &cli.Command{
	Name:  "list",
	Usage: "List all authorizations",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "tipset",
			Usage: "specify tipset to view block space usage of",
			Value: "@head",
		},
	},
	Action: func(cctx *cli.Context) error {
		api, closer, err := GetFullNodeAPI(cctx)
		if err != nil {
			return err
		}
		defer closer()

		ctx := ReqContext(cctx)

		ts, err := LoadTipSet(ctx, cctx, api)
		if err != nil {
			return err
		}

		infos, err := api.StateGovernorList(ctx, ts.Key())
		if err != nil {
			return err
		}

		for i, info := range infos {
			fmt.Printf("Governor %d: %s\n", i, info.Address)
			for _, au := range info.Authorities {
				actorName := builtin2.ActorNameByCode(au.ActorCodeID)
				methodNames := make([]string, len(au.Methods))
				for i, method := range au.Methods {
					methodNames[i] = fmt.Sprintf("%s(%d)", getMethodName(au.ActorCodeID, method), method)
				}
				fmt.Printf("\tActor: %-25s Methods: %s\n", actorName+",", strings.Join(methodNames, ", "))
			}
		}
		return nil
	},
}

//////////////////////////
//     gov knowledge
//////////////////////////
var govKnowledge = &cli.Command{
	Name:  "knowledge",
	Usage: "Manipulate knowledge fund params",
	Subcommands: []*cli.Command{
		govKnowledgeSetPayee,
	},
}

var govKnowledgeSetPayee = &cli.Command{
	Name:      "set-payee",
	Usage:     "Set knowledge fund payee address",
	ArgsUsage: "<newPayee>",
	Action: func(cctx *cli.Context) error {
		if cctx.Args().Len() != 1 {
			return fmt.Errorf("'set-payee' expects one arguments, new payee address")
		}

		api, acloser, err := GetFullNodeAPI(cctx)
		if err != nil {
			return err
		}
		defer acloser()
		ctx := ReqContext(cctx)

		newPayee, err := address.NewFromString(cctx.Args().Get(0))
		if err != nil {
			return err
		}

		fromAddr, err := parseFrom(cctx, ctx, api, true)
		if err != nil {
			return err
		}

		sp, err := actors.SerializeParams(&newPayee)
		if err != nil {
			return xerrors.Errorf("serializing params: %w", err)
		}

		if fmsig := cctx.String("msig"); fmsig == "" {
			return sendTransaction(cctx, ctx, api, fromAddr, builtin2.KnowledgeFundActorAddr, big.Zero(), builtin2.MethodsKnowledge.ChangePayee, sp)
		} else {
			governor, err := address.NewFromString(fmsig)
			if err != nil {
				return err
			}
			return sendProposal(cctx, ctx, api, governor, builtin2.KnowledgeFundActorAddr, fromAddr, big.Zero(), builtin2.MethodsKnowledge.ChangePayee, sp)
		}
	},
}

//////////////////////////
//     gov market
//////////////////////////
var govMarket = &cli.Command{
	Name:  "market",
	Usage: "Manipulate market actor",
	Subcommands: []*cli.Command{
		govMarketResetQuota,
		govMarketSetInitialQuota,
	},
}

var govMarketResetQuota = &cli.Command{
	Name:      "reset-quota",
	Usage:     "Reset power reward quota for specific PieceCID",
	ArgsUsage: "<PieceCID quota> <PieceCID quota> ...",
	Action: func(cctx *cli.Context) error {

		if !cctx.Args().Present() || cctx.Args().Len()%2 != 0 {
			return fmt.Errorf("'reset-quota' expects at least one <PieceCID number> pair")
		}

		api, acloser, err := GetFullNodeAPI(cctx)
		if err != nil {
			return err
		}
		defer acloser()

		ctx := ReqContext(cctx)

		// <PieceCID quota> pair
		params := &market2.ResetQuotasParams{}
		for i := 0; i < cctx.Args().Len()/2; i++ {
			pieceCid, err := cid.Decode(cctx.Args().Get(i * 2))
			if err != nil {
				return err
			}
			quota, err := strconv.ParseInt(cctx.Args().Get(i*2+1), 10, 64)
			if err != nil {
				return err
			}
			if quota < 0 {
				return fmt.Errorf("negative quota not allowed")
			}
			params.NewQuotas = append(params.NewQuotas, market2.NewQuota{PieceCID: pieceCid, Quota: quota})
		}

		fromAddr, err := parseFrom(cctx, ctx, api, true)
		if err != nil {
			return err
		}

		sp, err := actors.SerializeParams(params)
		if err != nil {
			return xerrors.Errorf("serializing params: %w", err)
		}

		if fmsig := cctx.String("msig"); fmsig == "" {
			return sendTransaction(cctx, ctx, api, fromAddr, builtin2.StorageMarketActorAddr, big.Zero(), builtin2.MethodsMarket.ResetQuotas, sp)
		} else {
			governor, err := address.NewFromString(fmsig)
			if err != nil {
				return err
			}
			return sendProposal(cctx, ctx, api, governor, builtin2.StorageMarketActorAddr, fromAddr, big.Zero(), builtin2.MethodsMarket.ResetQuotas, sp)
		}
	},
}

var govMarketSetInitialQuota = &cli.Command{
	Name:      "set-initial-quota",
	Usage:     "Set initial power reward quota for new pieces",
	ArgsUsage: "<quota>",
	Action: func(cctx *cli.Context) error {

		if cctx.Args().Len() != 1 {
			return fmt.Errorf("'set-initial-quota' expects one argument, initial quota number")
		}

		api, acloser, err := GetFullNodeAPI(cctx)
		if err != nil {
			return err
		}
		defer acloser()

		ctx := ReqContext(cctx)

		fromAddr, err := parseFrom(cctx, ctx, api, true)
		if err != nil {
			return err
		}

		quota, err := strconv.ParseInt(cctx.Args().First(), 10, 64)
		if err != nil {
			return err
		}
		if quota <= 0 {
			return fmt.Errorf("non-positive quota not allowed")
		}
		cbgQuota := cbg.CborInt(quota)
		sp, err := actors.SerializeParams(&cbgQuota)
		if err != nil {
			return xerrors.Errorf("serializing params: %w", err)
		}

		if fmsig := cctx.String("msig"); fmsig == "" {
			return sendTransaction(cctx, ctx, api, fromAddr, builtin2.StorageMarketActorAddr, big.Zero(), builtin2.MethodsMarket.SetInitialQuota, sp)
		} else {
			governor, err := address.NewFromString(fmsig)
			if err != nil {
				return err
			}
			return sendProposal(cctx, ctx, api, governor, builtin2.StorageMarketActorAddr, fromAddr, big.Zero(), builtin2.MethodsMarket.SetInitialQuota, sp)
		}
	},
}

//////////////////////////
//     gov expert
//////////////////////////
var govExpert = &cli.Command{
	Name:  "expert",
	Usage: "Manipulate expert actor",
	Subcommands: []*cli.Command{
		govExpertSetOwner,
		govExpertBlock,
	},
}
var govExpertSetOwner = &cli.Command{
	Name:      "set-owner",
	Usage:     "Set new owner address for an expert",
	ArgsUsage: "<expertAddress newOwnerAddress>",
	Action: func(cctx *cli.Context) error {
		if cctx.NArg() != 2 {
			return fmt.Errorf("'set-owner' expects two arguments, expert and new owner address")
		}

		api, acloser, err := GetFullNodeAPI(cctx)
		if err != nil {
			return err
		}
		defer acloser()
		ctx := ReqContext(cctx)

		expert, err := address.NewFromString(cctx.Args().Get(0))
		if err != nil {
			return err
		}

		newOwner, err := address.NewFromString(cctx.Args().Get(1))
		if err != nil {
			return err
		}

		fromAddr, err := parseFrom(cctx, ctx, api, true)
		if err != nil {
			return err
		}

		sp, err := actors.SerializeParams(&newOwner)
		if err != nil {
			return xerrors.Errorf("serializing params: %w", err)
		}

		if fmsig := cctx.String("msig"); fmsig == "" {
			return sendTransaction(cctx, ctx, api, fromAddr, expert, big.Zero(), builtin2.MethodsExpert.GovChangeOwner, sp)
		} else {
			governor, err := address.NewFromString(fmsig)
			if err != nil {
				return err
			}
			return sendProposal(cctx, ctx, api, governor, expert, fromAddr, big.Zero(), builtin2.MethodsExpert.GovChangeOwner, sp)
		}
	},
}

var govExpertBlock = &cli.Command{
	Name:      "block",
	Usage:     "Block an expert",
	ArgsUsage: "<expertAddress>",
	Action: func(cctx *cli.Context) error {
		if cctx.NArg() != 1 {
			return fmt.Errorf("'block' expects one argument, expert address")
		}

		api, acloser, err := GetFullNodeAPI(cctx)
		if err != nil {
			return err
		}
		defer acloser()
		ctx := ReqContext(cctx)

		expert, err := address.NewFromString(cctx.Args().First())
		if err != nil {
			return err
		}

		fromAddr, err := parseFrom(cctx, ctx, api, true)
		if err != nil {
			return err
		}

		if fmsig := cctx.String("msig"); fmsig == "" {
			return sendTransaction(cctx, ctx, api, fromAddr, expert, big.Zero(), builtin2.MethodsExpert.GovBlock, nil)
		} else {
			governor, err := address.NewFromString(fmsig)
			if err != nil {
				return err
			}
			return sendProposal(cctx, ctx, api, governor, expert, fromAddr, big.Zero(), builtin2.MethodsExpert.GovBlock, nil)
		}
	},
}

//////////////////////////
//     gov expertfund
//////////////////////////
var govExpertfund = &cli.Command{
	Name:  "expertfund",
	Usage: "Manipulate expertfund actor",
	Subcommands: []*cli.Command{
		govExpertfundSetThreshold,
	},
}
var govExpertfundSetThreshold = &cli.Command{
	Name:      "set-threshold",
	Usage:     "Set new threshold of data redundancy for valid expert acknowledgement",
	ArgsUsage: "<newThreshold>",
	Action: func(cctx *cli.Context) error {
		if cctx.NArg() != 1 {
			return fmt.Errorf("'set-threshold' expects one arguments, new threshold number")
		}

		api, acloser, err := GetFullNodeAPI(cctx)
		if err != nil {
			return err
		}
		defer acloser()
		ctx := ReqContext(cctx)

		threshold, err := strconv.ParseUint(cctx.Args().First(), 10, 64)
		if err != nil {
			return err
		}

		fromAddr, err := parseFrom(cctx, ctx, api, true)
		if err != nil {
			return err
		}

		sp, err := actors.SerializeParams(&expertfund.ChangeThresholdParams{DataStoreThreshold: threshold})
		if err != nil {
			return xerrors.Errorf("serializing params: %w", err)
		}

		if fmsig := cctx.String("msig"); fmsig == "" {
			return sendTransaction(cctx, ctx, api, fromAddr, builtin2.ExpertFundActorAddr, big.Zero(), builtin2.MethodsExpertFunds.ChangeThreshold, sp)
		} else {
			governor, err := address.NewFromString(fmsig)
			if err != nil {
				return err
			}
			return sendProposal(cctx, ctx, api, governor, builtin2.ExpertFundActorAddr, fromAddr, big.Zero(), builtin2.MethodsExpertFunds.ChangeThreshold, sp)
		}
	},
}

////////////////////
//     approve
////////////////////

var govApprove = &cli.Command{
	Name:      "approve",
	Usage:     "Approve a multisig message",
	ArgsUsage: "<messageId>",
	Action: func(cctx *cli.Context) error {
		if cctx.Args().Len() != 1 {
			return ShowHelp(cctx, fmt.Errorf("'approve' expects one argument, message ID"))
		}

		api, closer, err := GetFullNodeAPI(cctx)
		if err != nil {
			return err
		}
		defer closer()
		ctx := ReqContext(cctx)

		// msig
		var msig address.Address
		if fmsig := cctx.String("msig"); fmsig == "" {
			return fmt.Errorf("flag 'msig-governor' is required on command 'gov'")
		} else {
			msig, err = address.NewFromString(fmsig)
			if err != nil {
				return err
			}
		}

		// txid
		txid, err := strconv.ParseUint(cctx.Args().Get(0), 10, 64)
		if err != nil {
			return err
		}

		// from
		fromAddr, err := parseFrom(cctx, ctx, api, true)
		if err != nil {
			return err
		}

		// confirm tx
		{
			head, err := api.ChainHead(ctx)
			if err != nil {
				return err
			}

			act, err := api.StateGetActor(ctx, msig, head.Key())
			if err != nil {
				return err
			}

			store := adt.WrapStore(ctx, cbor.NewCborStore(blockstore.NewAPIBlockstore(api)))
			mstate, err := multisig.Load(store, act)
			if err != nil {
				return err
			}

			tx, err := mstate.PendingTxn(int64(txid))
			if err != nil {
				return err
			}
			targAct, err := api.StateGetActor(ctx, tx.To, types.EmptyTSK)
			if err != nil {
				return fmt.Errorf("failed to get actor %s (transaction.To): %w", tx.To, err)
			}
			// decode actor,method,param
			w := tabwriter.NewWriter(cctx.App.Writer, 8, 4, 2, ' ', 0)
			actorName := builtin2.ActorNameByCode(targAct.Code)
			method := stmgr.MethodsMap[targAct.Code][tx.Method]
			if tx.Method != 0 {
				fmt.Fprintf(w, "ID\tState\tApprovals\tTo\tValue\tMethod\tParams\n")
				ptyp := reflect.New(method.Params.Elem()).Interface().(cbg.CBORUnmarshaler)
				if err := ptyp.UnmarshalCBOR(bytes.NewReader(tx.Params)); err != nil {
					return xerrors.Errorf("failed to decode parameters of transaction %d: %w", txid, err)
				}

				b, err := json.Marshal(ptyp)
				if err != nil {
					return xerrors.Errorf("could not json marshal parameter type: %w", err)
				}

				paramStr := string(b)
				fmt.Fprintf(w, "%d\t%s\t%d\t%s(%s)\t%s\t%s(%d)\t%s\n", txid, "pending", len(tx.Approved), tx.To.String(), actorName,
					types.EPK(tx.Value), method.Name, tx.Method, paramStr)
			} else {
				return xerrors.Errorf("decode 'Send' transaction not implemented")
			}

			if err := w.Flush(); err != nil {
				return xerrors.Errorf("flushing output: %+v", err)
			}

			if !PromptConfirm("approve the proposal") {
				return nil
			}
		}

		return sendApprove(cctx, ctx, api, msig, txid, fromAddr)
	},
}

const (
	invalidMethodName = "<invalid>"
	unknownActor      = "<unknown actor>"
)

func getMethodName(code cid.Cid, num abi.MethodNum) string {
	for _, actor := range exported.BuiltinActors() {
		if actor.Code().Equals(code) {
			exports := actor.Exports()
			if len(exports) <= int(num) {
				return invalidMethodName
			}
			meth := exports[num]
			if meth == nil {
				return invalidMethodName
			}
			name := runtime.FuncForPC(reflect.ValueOf(meth).Pointer()).Name()
			name = strings.TrimSuffix(name, "-fm")
			lastDot := strings.LastIndexByte(name, '.')
			name = name[lastDot+1:]
			return name
		}
	}
	return unknownActor
}

func PromptConfirm(action string) bool {
	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("Confirm to %s (yes/no)?:", action)

	indata, err := reader.ReadBytes('\n')
	if err != nil {
		fmt.Println("error: ", err)
		return false
	}
	instr := strings.ToLower(strings.TrimSpace(string(indata)))
	return instr == "yes"
}

func PromptForGrantOrRevoke(out *gov2.GrantOrRevokeParams) error {

	reader := bufio.NewReader(os.Stdin)

	if len(gov2.GovernedActors) == 0 {
		return xerrors.Errorf("error: no actors")
	}

	var codes []cid.Cid
	for code := range gov2.GovernedActors {
		codes = append(codes, code)
	}
	sort.Slice(codes, func(i, j int) bool {
		return strings.Compare(codes[i].String(), codes[j].String()) < 0
	})

	fmt.Println()
	for i, code := range codes {
		actorName := builtin2.ActorNameByCode(code)
		fmt.Printf("\t%d - %s\n", i, actorName)
	}
	fmt.Printf("\t* - All above\n")

	var chosenCode cid.Cid
	// choose actor
	for {
		fmt.Printf("step 1: choose a actor (0-%d, or '*'):", len(codes)-1)
		indata, err := reader.ReadBytes('\n')
		if err != nil {
			return err
		}
		instr := strings.TrimSpace(string(indata))
		if instr == "" {
			continue
		}
		if instr == "*" {
			out.All = true
			return nil
		}
		aidx, err := strconv.Atoi(instr)
		if err != nil {
			fmt.Println("error: must input a integer or '*'")
			continue
		}
		if aidx < 0 || aidx >= len(codes) {
			fmt.Println("error: out of range")
			continue
		}

		chosenCode = codes[aidx]
		if len(gov2.GovernedActors[chosenCode]) == 0 {
			fmt.Printf("error: no methods on actor '%s'(%s)\n", builtin2.ActorNameByCode(chosenCode), chosenCode)
			continue
		}
		break
	}

	// choose methods
	fmt.Println()
	midx := -1
	var methods []abi.MethodNum
	for method := range gov2.GovernedActors[chosenCode] {
		midx++
		methods = append(methods, method)
		methodName := getMethodName(chosenCode, method)
		if methodName == invalidMethodName || methodName == unknownActor {
			return xerrors.Errorf("failed to get method name: code %s, method %d", chosenCode, method)
		}
		fmt.Printf("\t%d - %s\n", midx, methodName)
	}
	fmt.Printf("\t* - All above\n")

outer:
	for {
		fmt.Printf("step 2: choose one or more methods, space separated 0-%d, or '*':", midx)
		indata, err := reader.ReadBytes('\n')
		if err != nil {
			return err
		}

		instr := strings.TrimSpace(string(indata))
		if instr == "" {
			continue
		}

		auth := gov2.Authority{ActorCodeID: chosenCode}
		if instr == "*" {
			auth.All = true
		} else {
			fields := strings.Fields(instr)
			for _, field := range fields {
				midx, err := strconv.Atoi(field)
				if err != nil {
					fmt.Println("error: must input integer(s) or '*'")
					continue outer
				}
				if midx < 0 || midx >= len(methods) {
					fmt.Println("error: out of range")
					continue outer
				}
				auth.Methods = append(auth.Methods, methods[midx])
			}
		}
		out.Authorities = append(out.Authorities, auth)
		return nil
	}
}

func sendProposal(cctx *cli.Context, ctx context.Context, api api.FullNode,
	msigAddr, destAddr, fromAddr address.Address,
	value abi.TokenAmount,
	method abi.MethodNum,
	methodParam []byte,
) error {
	msgCid, err := api.MsigPropose(ctx, msigAddr, destAddr, value, fromAddr, uint64(method), methodParam)
	if err != nil {
		return err
	}

	fmt.Println("send proposal in message: ", msgCid)

	wait, err := api.StateWaitMsg(ctx, msgCid, uint64(cctx.Int("confidence")))
	if err != nil {
		return err
	}

	if wait.Receipt.ExitCode != 0 {
		return fmt.Errorf("proposal returned exit %d", wait.Receipt.ExitCode)
	}

	var retval msig2.ProposeReturn
	if err := retval.UnmarshalCBOR(bytes.NewReader(wait.Receipt.Return)); err != nil {
		return fmt.Errorf("failed to unmarshal propose return value: %w", err)
	}

	fmt.Printf("Transaction ID: %d\n", retval.TxnID)
	if retval.Applied {
		fmt.Printf("Transaction was executed during propose\n")
		fmt.Printf("Exit Code: %d\n", retval.Code)
		fmt.Printf("Return Value: %x\n", retval.Ret)
	}
	return nil
}

func sendApprove(cctx *cli.Context, ctx context.Context, api api.FullNode, msig address.Address, txid uint64, fromAddr address.Address) error {
	msgCid, err := api.MsigApprove(ctx, msig, txid, fromAddr)
	if err != nil {
		return err
	}

	fmt.Println("sent approval in message: ", msgCid)

	wait, err := api.StateWaitMsg(ctx, msgCid, uint64(cctx.Int("confidence")))
	if err != nil {
		return err
	}

	if wait.Receipt.ExitCode != 0 {
		return fmt.Errorf("approve returned exit %d", wait.Receipt.ExitCode)
	}

	fmt.Println("approve returned exit Ok")

	return nil
}

func sendTransaction(cctx *cli.Context, ctx context.Context, api api.FullNode,
	from, to address.Address, value abi.TokenAmount,
	method abi.MethodNum, methodParam []byte,
) error {
	smsg, err := api.MpoolPushMessage(ctx, &types.Message{
		To:     to,
		From:   from,
		Value:  value,
		Method: method,
		Params: methodParam,
	}, nil)
	if err != nil {
		return err
	}
	fmt.Println("wait message: ", smsg.Cid())

	wait, err := api.StateWaitMsg(ctx, smsg.Cid(), uint64(cctx.Int("confidence")))
	if err != nil {
		return err
	}

	if wait.Receipt.ExitCode != 0 {
		return fmt.Errorf("message exit %d", wait.Receipt.ExitCode)
	}
	return nil
}
