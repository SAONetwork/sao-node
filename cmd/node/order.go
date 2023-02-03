package main

import (
	"fmt"
	apiclient "sao-node/api/client"
	cliutil "sao-node/cmd"
	"strconv"

	"github.com/urfave/cli/v2"
	"golang.org/x/xerrors"
)

var ordersCmd = &cli.Command{
	Name:  "orders",
	Usage: "orders management",
	Subcommands: []*cli.Command{
		statusCmd,
		listCmd,
		fixCmd,
	},
}

var listCmd = &cli.Command{
	Name:  "list",
	Usage: "",
	Action: func(cctx *cli.Context) error {
		ctx := cctx.Context
		gatewayApi, closer, err := apiclient.NewGatewayApi(ctx, cliutil.Gateway, "DEFAULT_TOKEN")
		if err != nil {
			return err
		}
		defer closer()

		orders, err := gatewayApi.OrderList(ctx)
		if err != nil {
			return err
		}
		fmt.Println("Id State")
		for _, order := range orders {
			fmt.Printf("%d %s", order.OrderId, order.State.String())
			fmt.Println()
		}
		return nil
	},
}

var statusCmd = &cli.Command{
	Name:  "status",
	Usage: "",
	Action: func(cctx *cli.Context) error {
		ctx := cctx.Context
		gatewayApi, closer, err := apiclient.NewGatewayApi(ctx, cliutil.Gateway, "DEFAULT_TOKEN")
		if err != nil {
			return err
		}
		defer closer()

		if cctx.Args().Len() <= 0 {
			return xerrors.Errorf("missing order id parameter.")
		}
		orderId, err := strconv.ParseUint(cctx.Args().Get(0), 10, 64)
		if err != nil {
			return err
		}
		orderInfo, err := gatewayApi.OrderStatus(ctx, orderId)
		if err != nil {
			return err
		}
		fmt.Println("order id:", orderInfo.OrderId)
		fmt.Println("order state: ", orderInfo.State.String())
		return nil
	},
}

var fixCmd = &cli.Command {
	Name: "fix",
	Usage: "",
	Action: func(cctx *cli.Context) error {
		ctx := cctx.Context
		gatewayApi, closer, err := apiclient.NewGatewayApi(ctx, cliutil.Gateway, "DEFAULT_TOKEN")
		if err != nil {
			return err
		}
		defer closer()

		if cctx.Args().Len() <= 0 {
			return xerrors.Errorf("missing order id parameter.")
		}
		orderId, err := strconv.ParseUint(cctx.Args().Get(0), 10, 64)
		if err != nil {
			return err
		}

		orderInfo, err := gatewayApi.OrderFix(ctx, orderId)
		if err != nil {
			return err
		}
		fmt.Println("order id:", orderInfo.OrderId)
		fmt.Println("order state: ", orderInfo.State.String())
		return nil	
	}
}
