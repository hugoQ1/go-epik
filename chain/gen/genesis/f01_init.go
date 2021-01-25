package genesis

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/specs-actors/v2/actors/builtin"
	init_ "github.com/filecoin-project/specs-actors/v2/actors/builtin/init"
	"github.com/filecoin-project/specs-actors/v2/actors/util/adt"

	cbor "github.com/ipfs/go-ipld-cbor"
	cbg "github.com/whyrusleeping/cbor-gen"
	"golang.org/x/xerrors"

	"github.com/EpiK-Protocol/go-epik/chain/types"
	"github.com/EpiK-Protocol/go-epik/genesis"
	bstore "github.com/EpiK-Protocol/go-epik/lib/blockstore"
)

func SetupInitActor(bs bstore.Blockstore, tpl genesis.Template) (int64, *types.Actor, map[address.Address]address.Address, error) {
	initialActors := append(tpl.Accounts,
		tpl.FoundationAccountActor,
		tpl.FundraisingAccountActor,
		tpl.TeamAccountActor,
		tpl.DefaultGovernorActor,
		tpl.DefaultExpertActor,
		tpl.DefaultKgFundPayeeActor,
	)
	if len(initialActors) > MaxAccounts {
		return 0, nil, nil, xerrors.New("too many initial actors")
	}

	var ias init_.State
	ias.NextID = MinerStart
	ias.NetworkName = tpl.NetworkName

	store := adt.WrapStore(context.TODO(), cbor.NewCborStore(bs))
	amap := adt.MakeEmptyMap(store)

	keyToId := map[address.Address]address.Address{}
	counter := int64(AccountStart)

	for _, a := range initialActors {
		if a.Type == genesis.TMultisig {
			var ainfo genesis.MultisigMeta
			if err := json.Unmarshal(a.Meta, &ainfo); err != nil {
				return 0, nil, nil, xerrors.Errorf("unmarshaling account meta: %w", err)
			}
			for _, e := range ainfo.Signers {

				if _, ok := keyToId[e]; ok {
					continue
				}

				fmt.Printf("init set %s t0%d (msig)\n", e, counter)

				value := cbg.CborInt(counter)
				if err := amap.Put(abi.AddrKey(e), &value); err != nil {
					return 0, nil, nil, err
				}
				counter = counter + 1
				var err error
				keyToId[e], err = address.NewIDAddress(uint64(value))
				if err != nil {
					return 0, nil, nil, err
				}

			}
			// Need to add actors for all multisigs too
			continue
		}

		if a.Type != genesis.TAccount {
			return 0, nil, nil, xerrors.Errorf("unsupported account type: %s", a.Type)
		}

		var ainfo genesis.AccountMeta
		if err := json.Unmarshal(a.Meta, &ainfo); err != nil {
			return 0, nil, nil, xerrors.Errorf("unmarshaling account meta: %w", err)
		}

		if _, ok := keyToId[ainfo.Owner]; ok {
			continue
		}

		fmt.Printf("init set %s t0%d\n", ainfo.Owner, counter)

		value := cbg.CborInt(counter)
		if err := amap.Put(abi.AddrKey(ainfo.Owner), &value); err != nil {
			return 0, nil, nil, err
		}
		counter = counter + 1

		var err error
		keyToId[ainfo.Owner], err = address.NewIDAddress(uint64(value))
		if err != nil {
			return 0, nil, nil, err
		}
	}

	amapaddr, err := amap.Root()
	if err != nil {
		return 0, nil, nil, err
	}
	ias.AddressMap = amapaddr

	statecid, err := store.Put(store.Context(), &ias)
	if err != nil {
		return 0, nil, nil, err
	}

	act := &types.Actor{
		Code: builtin.InitActorCodeID,
		Head: statecid,
	}

	return counter, act, keyToId, nil
}
