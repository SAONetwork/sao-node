package main

import (
	"fmt"
	"os"

	apiclient "github.com/SaoNetwork/sao-node/api/client"
	cliutil "github.com/SaoNetwork/sao-node/cmd"
	"github.com/SaoNetwork/sao-node/types"

	"github.com/filecoin-project/lotus/lib/tablewriter"
	"github.com/urfave/cli/v2"
)

var ordersCmd = &cli.Command{
	Name:  "orders",
	Usage: "orders management",
	Subcommands: []*cli.Command{
		orderStatusCmd,
		orderListCmd,
		// orderFixCmd,
	},
}

var orderListCmd = &cli.Command{
	Name:  "list",
	Usage: "List orders",
	Action: func(cctx *cli.Context) error {
		ctx := cctx.Context
		gatewayApi, closer, err := apiclient.NewNodeApi(ctx, NodeApi, cliutil.ApiToken)
		if err != nil {
			return err
		}
		defer closer()

		orders, err := gatewayApi.OrderList(ctx)
		if err != nil {
			return err
		}

		tw := tablewriter.New(
			tablewriter.Col("Id"),
			tablewriter.Col("OrderId"),
			tablewriter.Col("State"),
		)
		for _, order := range orders {
			tw.Write(map[string]interface{}{
				"Id":      order.DataId,
				"OrderId": order.OrderId,
				"State":   order.State,
			})
		}
		return tw.Flush(os.Stdout)
	},
}

var orderStatusCmd = &cli.Command{
	Name:  "status",
	Usage: "",
	Action: func(cctx *cli.Context) error {
		ctx := cctx.Context
		apiClient, closer, err := cliutil.GetNodeApi(cctx, cctx.String(FlagStorageRepo), NodeApi, cliutil.ApiToken)
		if err != nil {
			return err
		}
		defer closer()

		if cctx.Args().Len() <= 0 {
			return types.Wrapf(types.ErrInvalidParameters, "missing proposal id parameter.")
		}
		dataId := cctx.Args().Get(0)
		orderInfo, err := apiClient.OrderStatus(ctx, dataId)
		if err != nil {
			return err
		}
		fmt.Println("Id: ", orderInfo.DataId)
		fmt.Println("OrderId: ", orderInfo.OrderId)
		fmt.Println("State: ", orderInfo.State.String())
		return nil
	},
}
