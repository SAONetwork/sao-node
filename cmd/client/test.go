package main

import (
	apiclient "sao-storage-node/api/client"
	saoclient "sao-storage-node/client"

	"github.com/urfave/cli/v2"
)

var testCmd = &cli.Command{
	Name: "test",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:     "gateway",
			Value:    "http://127.0.0.1:8888/rpc/v0",
			EnvVars:  []string{"SAO_GATEWAY_API"},
			Required: false,
		},
	},
	Action: func(cctx *cli.Context) error {
		ctx := cctx.Context
		gateway := cctx.String("gateway")
		gatewayApi, closer, err := apiclient.NewGatewayApi(ctx, gateway, nil)
		if err != nil {
			return err
		}
		defer closer()

		client := saoclient.NewSaoClient(gatewayApi)
		resp, err := client.Test(ctx)
		if err != nil {
			return err
		}
		log.Info(resp)
		return nil
	},
}
