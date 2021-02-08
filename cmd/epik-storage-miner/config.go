package main

import (
	"fmt"

	"github.com/urfave/cli/v2"

	"github.com/EpiK-Protocol/go-epik/node/config"
)

var configCmd = &cli.Command{
	Name:  "config",
	Usage: "Output default configuration",
	Action: func(cctx *cli.Context) error {
		comm, err := config.ConfigComment(config.DefaultStorageMiner())
		if err != nil {
			return err
		}
		fmt.Println(string(comm))
		return nil
	},
}
