package main

import (
	"fmt"
	"os"

	"github.com/EpiK-Protocol/go-epik/build"
	logging "github.com/ipfs/go-log/v2"
	"github.com/urfave/cli/v2"
)

var log = logging.Logger("expressman")

func main() {
	if err := logging.SetLogLevel("*", "info"); err != nil {
		log.Fatal(err)
	}
	log.Info("Starting expressman", " v", build.UserVersion())

	app := &cli.App{
		Name:    "epik-expressman",
		Usage:   "Upload/download files in batch.",
		Version: build.UserVersion(),
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "repo",
				EnvVars: []string{"EPIK_PATH"},
				Hidden:  true,
				Value:   "~/.epik",
			},
			&cli.StringFlag{
				Name:    "miner-repo",
				Aliases: []string{"storagerepo"},
				EnvVars: []string{"EPIK_MINER_PATH", "EPIK_STORAGE_PATH"},
				Value:   "~/.epikminer",
				Usage:   fmt.Sprintf("Specify miner repo path. flag(%s) and env(EPIK_STORAGE_PATH) are DEPRECATION, will REMOVE SOON", "storagerepo"),
			},
		},
		Commands: []*cli.Command{
			uploadCmd,
			downloadCmd,
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
