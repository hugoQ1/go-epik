package main

import (
	"context"
	"os"

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

const FlagStorageRepo = "storagerepo"

func main() {
	epiklog.SetupLogLevels()

	local := []*cli.Command{
		actorCmd,
		storageDealsCmd,
		retrievalDealsCmd,
		infoCmd,
		initCmd,
		rewardsCmd,
		runCmd,
		stopCmd,
		sectorsCmd,
		storageCmd,
		workersCmd,
		provingCmd,
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
		Name:                 "epik-storage-miner",
		Usage:                "EpiK decentralized storage network storage miner",
		Version:              build.UserVersion(),
		EnableBashCompletion: true,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "actor",
				Value:   "",
				Usage:   "specify other actor to check state for (read only)",
				Aliases: []string{"a"},
			},
			&cli.StringFlag{
				Name:    "repo",
				EnvVars: []string{"EPIK_PATH"},
				Hidden:  true,
				Value:   "~/.epik", // TODO: Consider XDG_DATA_HOME
			},
			&cli.StringFlag{
				Name:    FlagStorageRepo,
				EnvVars: []string{"EPIK_STORAGE_PATH"},
				Value:   "~/.epikstorage", // TODO: Consider XDG_DATA_HOME
			},
		},

		Commands: append(local, lcli.CommonCommands...),
	}
	app.Setup()
	app.Metadata["repoType"] = repo.StorageMiner

	if err := app.Run(os.Args); err != nil {
		log.Warnf("%+v", err)
		os.Exit(1)
	}
}

func getActorAddress(ctx context.Context, nodeAPI api.StorageMiner, overrideMaddr string) (maddr address.Address, err error) {
	if overrideMaddr != "" {
		maddr, err = address.NewFromString(overrideMaddr)
		if err != nil {
			return maddr, err
		}
	}

	maddr, err = nodeAPI.ActorAddress(ctx)
	if err != nil {
		return maddr, xerrors.Errorf("getting actor address: %w", err)
	}

	return maddr, nil
}
