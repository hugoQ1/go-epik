package main

import (
	"context"

	"github.com/urfave/cli/v2"
	"go.opencensus.io/trace"

	"github.com/EpiK-Protocol/go-epik/build"
	lcli "github.com/EpiK-Protocol/go-epik/cli"
	"github.com/EpiK-Protocol/go-epik/lib/epiklog"
	"github.com/EpiK-Protocol/go-epik/lib/tracing"
	"github.com/EpiK-Protocol/go-epik/node/repo"
)

var AdvanceBlockCmd *cli.Command

func main() {
	build.RunningNodeType = build.NodeFull

	epiklog.SetupLogLevels()

	local := []*cli.Command{
		DaemonCmd,
		backupCmd,
	}
	if AdvanceBlockCmd != nil {
		local = append(local, AdvanceBlockCmd)
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
	ctx, span := trace.StartSpan(context.Background(), "/cli")
	defer span.End()

	app := &cli.App{
		Name:                 "epik",
		Usage:                "EpiK decentralized storage network client",
		Version:              build.UserVersion(),
		EnableBashCompletion: true,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "repo",
				EnvVars: []string{"EPIK_PATH"},
				Hidden:  true,
				Value:   "~/.epik", // TODO: Consider XDG_DATA_HOME
			},
		},

		Commands: append(local, lcli.Commands...),
	}
	app.Setup()
	app.Metadata["traceContext"] = ctx
	app.Metadata["repoType"] = repo.FullNode

	lcli.RunApp(app)
}
