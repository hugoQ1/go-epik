package main

import (
	"os"

	logging "github.com/ipfs/go-log/v2"
	"github.com/urfave/cli/v2"

	"github.com/EpiK-Protocol/go-epik/build"
)

var log = logging.Logger("epik-shed")

func main() {
	logging.SetLogLevel("*", "INFO")

	local := []*cli.Command{
		base32Cmd,
		base16Cmd,
		bitFieldCmd,
		keyinfoCmd,
		noncefix,
		bigIntParseCmd,
		staterootStatsCmd,
		importCarCmd,
		commpToCidCmd,
		fetchParamCmd,
		proofsCmd,
		verifRegCmd,
	}

	app := &cli.App{
		Name:     "epik-shed",
		Usage:    "A place for all the epik tools",
		Version:  build.BuildVersion,
		Commands: local,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "repo",
				EnvVars: []string{"EPIK_PATH"},
				Hidden:  true,
				Value:   "~/.epik", // TODO: Consider XDG_DATA_HOME
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Warnf("%+v", err)
		os.Exit(1)
		return
	}
}
