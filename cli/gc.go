package cli

import (
	"github.com/urfave/cli/v2"
)

var gcCmd = &cli.Command{
	Name:   "gc",
	Usage:  "db gc",
	Hidden: true,
	Flags: []cli.Flag{
		&cli.BoolFlag{
			Name:  "force",
			Usage: "force to gc db",
			Value: false,
		},
	},
	Action: func(cctx *cli.Context) error {
		if !cctx.Bool("force") {
			return cli.ShowCommandHelp(cctx, cctx.Command.Name)
		}
		api, closer, err := GetFullNodeAPI(cctx)
		if err != nil {
			return err
		}
		defer closer()
		ctx := ReqContext(cctx)

		return api.GC(ctx)
	},
}
