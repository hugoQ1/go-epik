package main

import (
	"context"
	"fmt"

	logging "github.com/ipfs/go-log/v2"
	"github.com/urfave/cli/v2"
	"go.opencensus.io/trace"
	"golang.org/x/xerrors"

	"github.com/EpiK-Protocol/go-epik/api"
	"github.com/EpiK-Protocol/go-epik/build"
	lcli "github.com/EpiK-Protocol/go-epik/cli"
	"github.com/EpiK-Protocol/go-epik/lib/epiklog"
	"github.com/EpiK-Protocol/go-epik/lib/tracing"
	"github.com/EpiK-Protocol/go-epik/node/repo"
	"github.com/filecoin-project/go-address"
)

var log = logging.Logger("main")

const FlagMinerRepo = "miner-repo"

// TODO remove after deprecation period
const FlagMinerRepoDeprecation = "storagerepo"

func main() {
	build.RunningNodeType = build.NodeMiner

	epiklog.SetupLogLevels()

	local := []*cli.Command{
		initCmd,
		runCmd,
		stopCmd,
		configCmd,
		backupCmd,
		lcli.WithCategory("chain", actorCmd),
		lcli.WithCategory("chain", infoCmd),
		lcli.WithCategory("market", storageDealsCmd),
		lcli.WithCategory("market", retrievalDealsCmd),
		lcli.WithCategory("market", dataTransfersCmd),
		lcli.WithCategory("storage", sectorsCmd),
		lcli.WithCategory("storage", provingCmd),
		lcli.WithCategory("storage", storageCmd),
		lcli.WithCategory("storage", sealingCmd),
		lcli.WithCategory("retrieval", piecesCmd),
	}
	jaeger := tracing.SetupJaegerTracing("epik")
	defer func() {
		if jaeger != nil {
			jaeger.Flush()
		}
	}()

	for _, cmd := range local {
		cmd := cmd
		originBefore := cmd.Before
		cmd.Before = func(cctx *cli.Context) error {
			trace.UnregisterExporter(jaeger)
			jaeger = tracing.SetupJaegerTracing("epik/" + cmd.Name)

			if originBefore != nil {
				return originBefore(cctx)
			}
			return nil
		}
	}

	app := &cli.App{
		Name:                 "epik-miner",
		Usage:                "Filecoin decentralized storage network miner",
		Version:              build.UserVersion(),
		EnableBashCompletion: true,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "actor",
				Value:   "",
				Usage:   "specify other actor to check state for (read only)",
				Aliases: []string{"a"},
			},
			&cli.BoolFlag{
				Name: "color",
			},
			&cli.StringFlag{
				Name:    "repo",
				EnvVars: []string{"EPIK_PATH"},
				Hidden:  true,
				Value:   "~/.epik", // TODO: Consider XDG_DATA_HOME
			},
			&cli.StringFlag{
				Name:    FlagMinerRepo,
				Aliases: []string{FlagMinerRepoDeprecation},
				EnvVars: []string{"EPIK_MINER_PATH", "EPIK_STORAGE_PATH"},
				Value:   "~/.epikminer", // TODO: Consider XDG_DATA_HOME
				Usage:   fmt.Sprintf("Specify miner repo path. flag(%s) and env(EPIK_STORAGE_PATH) are DEPRECATION, will REMOVE SOON", FlagMinerRepoDeprecation),
			},
		},

		Commands: append(local, lcli.CommonCommands...),
	}
	app.Setup()
	app.Metadata["repoType"] = repo.StorageMiner

	lcli.RunApp(app)
}

func getActorAddress(ctx context.Context, nodeAPI api.StorageMiner, overrideMaddr string) (maddr address.Address, err error) {
	if overrideMaddr != "" {
		maddr, err = address.NewFromString(overrideMaddr)
		if err != nil {
			return maddr, err
		}
		return
	}

	maddr, err = nodeAPI.ActorAddress(ctx)
	if err != nil {
		return maddr, xerrors.Errorf("getting actor address: %w", err)
	}

	return maddr, nil
}
