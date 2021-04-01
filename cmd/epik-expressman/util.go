package main

import (
	"context"

	lapi "github.com/EpiK-Protocol/go-epik/api"
	"github.com/EpiK-Protocol/go-epik/chain/types"
	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/specs-actors/v2/actors/builtin/expert"
	"golang.org/x/xerrors"
)

func findExpert(ctx context.Context, owner address.Address, api lapi.FullNode) (address.Address, error) {
	ida, err := api.StateLookupID(ctx, owner, types.EmptyTSK)
	if err != nil {
		return address.Undef, err
	}

	experts, err := api.StateListExperts(ctx, types.EmptyTSK)
	if err != nil {
		return address.Undef, err
	}
	for _, exp := range experts {
		ei, err := api.StateExpertInfo(ctx, exp, types.EmptyTSK)
		if err != nil {
			return address.Undef, err
		}
		if ei.Owner == ida {
			if ei.Status != expert.ExpertStateNormal {
				return address.Undef, xerrors.Errorf("expert not in normal state: %s, %d", exp, ei.Status)
			}
			return exp, nil
		}
	}
	return address.Undef, xerrors.Errorf("no expert found with current used wallet address %s", owner)
}
