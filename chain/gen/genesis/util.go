package genesis

import (
	"context"
	"fmt"

	"github.com/EpiK-Protocol/go-epik/build"
	"github.com/EpiK-Protocol/go-epik/genesis"
	"github.com/filecoin-project/go-state-types/network"
	"github.com/ipfs/go-cid"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	cbg "github.com/whyrusleeping/cbor-gen"
	"golang.org/x/xerrors"

	"github.com/EpiK-Protocol/go-epik/chain/actors"
	"github.com/EpiK-Protocol/go-epik/chain/types"
	"github.com/EpiK-Protocol/go-epik/chain/vm"

	"github.com/filecoin-project/go-commp-utils/ffiwrapper"
)

func mustEnc(i cbg.CBORMarshaler) []byte {
	enc, err := actors.SerializeParams(i)
	if err != nil {
		panic(err) // ok
	}
	return enc
}

func doExecValue(ctx context.Context, vm *vm.VM, to, from address.Address, value types.BigInt, method abi.MethodNum, params []byte) ([]byte, error) {
	act, err := vm.StateTree().GetActor(from)
	if err != nil {
		return nil, xerrors.Errorf("doExec failed to get from actor (%s): %w", from, err)
	}

	ret, err := vm.ApplyImplicitMessage(ctx, &types.Message{
		To:       to,
		From:     from,
		Method:   method,
		Params:   params,
		GasLimit: 1_000_000_000_000_000,
		Value:    value,
		Nonce:    act.Nonce,
	})
	if err != nil {
		return nil, xerrors.Errorf("doExec apply message failed: %w", err)
	}

	if ret.ExitCode != 0 {
		return nil, xerrors.Errorf("failed to call method: %w", ret.ActorErr)
	}

	return ret.Return, nil
}

// TODO: Get from build
// TODO: make a list/schedule of these.
var GenesisNetworkVersion = func() network.Version {
	return build.NewestNetworkVersion
}()

func genesisNetworkVersion(context.Context, abi.ChainEpoch) network.Version { // TODO: Get from build/
	return GenesisNetworkVersion // TODO: Get from build/
} // TODO: Get from build/

// UnsealedCID
func GeneratePaddedPresealFileCID(pt abi.RegisteredSealProof) (cid.Cid, error) {
	ssize, err := pt.SectorSize()
	if err != nil {
		return cid.Undef, err
	}
	unpadded := abi.PaddedPieceSize(ssize).Unpadded()
	fmt.Printf("generate preseal file piece cid, to read %d bytes\n", unpadded)

	// param proofType could be arbitrary, cause CommD is sector-size independent
	return ffiwrapper.GeneratePieceCIDFromFile(pt, genesis.NewZeroPaddingPresealFileReader(), unpadded)
}
