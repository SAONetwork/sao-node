package main

import (
	saoclient "sao-storage-node/client"

	"github.com/urfave/cli/v2"
)

var TestCmd = &cli.Command{
	Name: "test",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:     "gateway",
			EnvVars:  []string{"SAO_GATEWAY_API"},
			Required: false,
		},
	},
	Action: func(cctx *cli.Context) error {
		ctx := cctx.Context
		gateway := cctx.String("gateway")

		client := saoclient.NewSaoClient(ctx, gateway)
		resp, err := client.Test(ctx)
		if err != nil {
			return err
		}
		log.Info(resp)
		return nil
	},
}
