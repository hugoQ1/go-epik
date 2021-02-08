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
	"github.com/EpiK-Protocol/go-epik/api/apibstore"
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
	"github.com/filecoin-project/specs-actors/v2/actors/builtin/exported"
	gov2 "github.com/filecoin-project/specs-actors/v2/actors/builtin/govern"
	market2 "github.com/filecoin-project/specs-actors/v2/actors/builtin/market"
	msig2 "github.com/filecoin-project/specs-actors/v2/actors/builtin/multisig"
)

var govCmd = &cli.Command{
	Name:  "gov",
	Usage: "Govern epik network",
	// Flags: []cli.Flag{
	// 	&cli.BoolFlag{
	// 		Name:  "really-do-it",
	// 		Usage: "Actually send transaction performing the action",
	// 		Value: false,
	// 	},
	// },
	Subcommands: []*cli.Command{
		govPropose,
		govApproveTx,
		govListGovernorsCmd,
	},
	// Before: func(cctx *cli.Context) error {
	// 	if !cctx.Bool("really-do-it") {
	// 		return fmt.Errorf("Pass --really-do-it to actually execute this action")
	// 	}
	// 	return nil
	// },
}

////////////////////
//     propose
////////////////////

var govPropose = &cli.Command{
	Name:  "propose",
	Usage: "Propose a governance transation",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:    "from",
			Usage:   "Specify the proposer address, otherwise use the default wallet address",
			Aliases: []string{"f"},
		},
	},
	Subcommands: []*cli.Command{
		govProposeGrant,
		govProposeRevoke,
		govProposeSetKnowledgePayee,
		govProposeResetPieceQuota,
		govProposeSetInitialQuota,
	},
}

var govProposeGrant = &cli.Command{
	Name:      "grant",
	Usage:     "Propose granting priviledges to governor",
	ArgsUsage: "[targetAddress]",
	Action: func(cctx *cli.Context) error {
		if !cctx.Args().Present() {
			return ShowHelp(cctx, fmt.Errorf("'grant' expects one argument, target governor"))
		}

		api, closer, err := GetFullNodeAPI(cctx)
		if err != nil {
			return err
		}
		defer closer()

		ctx := ReqContext(cctx)

		// target
		target, err := address.NewFromString(cctx.Args().First())
		if err != nil {
			return fmt.Errorf("failed to parse target address: %w", err)
		}
		ida, err := api.StateLookupID(ctx, target, types.EmptyTSK)
		if err != nil {
			return fmt.Errorf("failed to lookup id address for %s", target)
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

var govProposeRevoke = &cli.Command{
	Name:      "revoke",
	Usage:     "Propose revoking priviledges from governor",
	ArgsUsage: "[targetAddress]",
	Action: func(cctx *cli.Context) error {
		if !cctx.Args().Present() {
			return ShowHelp(cctx, fmt.Errorf("'revoke' expects one argument, target governor"))
		}

		api, closer, err := GetFullNodeAPI(cctx)
		if err != nil {
			return err
		}
		defer closer()

		ctx := ReqContext(cctx)

		// target
		target, err := address.NewFromString(cctx.Args().First())
		if err != nil {
			return fmt.Errorf("failed to parse target address: %w", err)
		}
		ida, err := api.StateLookupID(ctx, target, types.EmptyTSK)
		if err != nil {
			return fmt.Errorf("failed to lookup id address for %s", target)
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

var govProposeSetKnowledgePayee = &cli.Command{
	Name:      "set-knowledge-payee",
	Usage:     "Set knowledge fund payee address",
	ArgsUsage: "[governorAddress] [newPayeeAddress]",
	Action: func(cctx *cli.Context) error {
		if cctx.Args().Len() != 2 {
			return fmt.Errorf("expect two arguments, governor and new payee address")
		}

		// governor(msig)
		governor, err := address.NewFromString(cctx.Args().Get(0))
		if err != nil {
			return err
		}
		newPayee, err := address.NewFromString(cctx.Args().Get(1))
		if err != nil {
			return err
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

		sp, err := actors.SerializeParams(&newPayee)
		if err != nil {
			return xerrors.Errorf("serializing params: %w", err)
		}

		return sendProposal(cctx, ctx, api, governor, builtin2.KnowledgeFundActorAddr, fromAddr, big.Zero(), builtin2.MethodsKnowledge.ChangePayee, sp)
	},
}

var govProposeResetPieceQuota = &cli.Command{
	Name:      "reset-piece-quota",
	Usage:     "Reset reward quota for PieceCID",
	ArgsUsage: "[governorAddress] [PieceCID number] [PieceCID number] [...]",
	Action: func(cctx *cli.Context) error {

		if cctx.Args().Len() < 3 || cctx.Args().Len()%2 != 1 {
			return fmt.Errorf("expect governor address and at least one [PieceCID number] pair")
		}

		api, acloser, err := GetFullNodeAPI(cctx)
		if err != nil {
			return err
		}
		defer acloser()

		ctx := ReqContext(cctx)

		// governor(msig)
		governor, err := address.NewFromString(cctx.Args().Get(0))
		if err != nil {
			return err
		}
		// [PieceCID number] pair
		params := &market2.ResetQuotasParams{}
		for i := 0; i < cctx.Args().Len()/2; i++ {
			pieceCid, err := cid.Decode(cctx.Args().Get(i*2 + 1))
			if err != nil {
				return err
			}
			quota, err := strconv.ParseInt(cctx.Args().Get(i*2+2), 10, 64)
			if err != nil {
				return err
			}
			if quota < 0 {
				return fmt.Errorf("negatvie quota not allowed")
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

		return sendProposal(cctx, ctx, api, governor, builtin2.StorageMarketActorAddr, fromAddr, big.Zero(), builtin2.MethodsMarket.ResetQuotas, sp)
	},
}

var govProposeSetInitialQuota = &cli.Command{
	Name:      "set-initial-quota",
	Usage:     "Set initial quota for new piece",
	ArgsUsage: "[governorAddress] [quota]",
	Action: func(cctx *cli.Context) error {

		if cctx.Args().Len() != 2 {
			return fmt.Errorf("expect two arguments, governor address and initial quota number")
		}

		api, acloser, err := GetFullNodeAPI(cctx)
		if err != nil {
			return err
		}
		defer acloser()

		ctx := ReqContext(cctx)

		// governor(msig)
		governor, err := address.NewFromString(cctx.Args().Get(0))
		if err != nil {
			return err
		}

		fromAddr, err := parseFrom(cctx, ctx, api, true)
		if err != nil {
			return err
		}

		quota, err := strconv.ParseInt(cctx.Args().Get(1), 10, 64)
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

		return sendProposal(cctx, ctx, api, governor, builtin2.StorageMarketActorAddr, fromAddr, big.Zero(), builtin2.MethodsMarket.SetInitialQuota, sp)
	},
}

var govListGovernorsCmd = &cli.Command{
	Name:  "list-governors",
	Usage: "List all governors",
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

////////////////////
//     approve
////////////////////

var govApproveTx = &cli.Command{
	Name:      "approve",
	Usage:     "Approve a governance transation",
	ArgsUsage: "[multisigAddress] [txId]",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:    "from",
			Usage:   "Specify the approver address, otherwise use the default wallet address",
			Aliases: []string{"f"},
		},
	},
	Action: func(cctx *cli.Context) error {
		if cctx.Args().Len() != 2 {
			return ShowHelp(cctx, fmt.Errorf("'approve' expects two argument, multisig address and transaction ID"))
		}

		// msig
		msig, err := address.NewFromString(cctx.Args().Get(0))
		if err != nil {
			return err
		}

		// txid
		txid, err := strconv.ParseUint(cctx.Args().Get(1), 10, 64)
		if err != nil {
			return err
		}

		api, closer, err := GetFullNodeAPI(cctx)
		if err != nil {
			return err
		}
		defer closer()
		ctx := ReqContext(cctx)

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

			store := adt.WrapStore(ctx, cbor.NewCborStore(apibstore.NewAPIBlockstore(api)))
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

func parseFrom(cctx *cli.Context, ctx context.Context, api api.FullNode, useDef bool) (address.Address, error) {
	from := cctx.String("from")
	if from == "" {
		if !useDef {
			return address.Undef, fmt.Errorf("from not set")
		}
		return api.WalletDefaultAddress(ctx)
	}

	return address.NewFromString(from)
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
