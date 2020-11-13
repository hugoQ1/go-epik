package remotewallet

import (
	"context"

	"go.uber.org/fx"
	"golang.org/x/xerrors"

	"github.com/EpiK-Protocol/go-epik/api"
	"github.com/EpiK-Protocol/go-epik/api/client"
	cliutil "github.com/EpiK-Protocol/go-epik/cli/util"
	"github.com/EpiK-Protocol/go-epik/node/modules/helpers"
)

type RemoteWallet struct {
	api.WalletAPI
}

func SetupRemoteWallet(info string) func(mctx helpers.MetricsCtx, lc fx.Lifecycle) (*RemoteWallet, error) {
	return func(mctx helpers.MetricsCtx, lc fx.Lifecycle) (*RemoteWallet, error) {
		ai := cliutil.ParseApiInfo(info)

		url, err := ai.DialArgs()
		if err != nil {
			return nil, err
		}

		wapi, closer, err := client.NewWalletRPC(mctx, url, ai.AuthHeader())
		if err != nil {
			return nil, xerrors.Errorf("creating jsonrpc client: %w", err)
		}

		lc.Append(fx.Hook{
			OnStop: func(ctx context.Context) error {
				closer()
				return nil
			},
		})

		return &RemoteWallet{wapi}, nil
	}
}

func (w *RemoteWallet) Get() api.WalletAPI {
	if w == nil {
		return nil
	}

	return w
}
