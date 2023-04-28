package main

import (
	"fmt"
	"os"
	cliutil "sao-node/cmd"

	"github.com/filecoin-project/lotus/lib/tablewriter"
	"github.com/ipfs/go-cid"

	"github.com/urfave/cli/v2"
)

var shardsCmd = &cli.Command{
	Name:  "shards",
	Usage: "shards management",
	Subcommands: []*cli.Command{
		shardStatusCmd,
		shardListCmd,
		// shardFixCmd,
	},
}

var shardStatusCmd = &cli.Command{
	Name:  "status",
	Usage: "show specified shard status",
	Flags: []cli.Flag{
		&cli.Uint64Flag{
			Name:     "orderId",
			Required: true,
		},
		&cli.StringFlag{
			Name:     "cid",
			Required: true,
		},
	},
	Action: func(cctx *cli.Context) error {
		ctx := cctx.Context
		orderId := cctx.Uint64("orderId")
		shardCidStr := cctx.String("cid")
		shardCid, err := cid.Decode(shardCidStr)
		if err != nil {
			return err
		}

		apiClient, closer, err := cliutil.GetNodeApi(cctx, cctx.String(FlagStorageRepo), NodeApi, cliutil.ApiToken)
		if err != nil {
			return err
		}
		defer closer()
		shardInfo, err := apiClient.ShardStatus(ctx, orderId, shardCid)
		if err != nil {
			return err
		}
		fmt.Println("OrderId: ", orderId)
		fmt.Println("Cid: ", shardCid)
		fmt.Println("State: ", shardInfo.State)

		return nil
	},
}

var shardListCmd = &cli.Command{
	Name:  "list",
	Usage: "List shards",
	Action: func(cctx *cli.Context) error {
		ctx := cctx.Context
		apiClient, closer, err := cliutil.GetNodeApi(cctx, cctx.String(FlagStorageRepo), NodeApi, cliutil.ApiToken)
		if err != nil {
			return err
		}
		defer closer()

		shards, err := apiClient.ShardList(ctx)
		if err != nil {
			return err
		}

		tw := tablewriter.New(
			tablewriter.Col("OrderId"),
			tablewriter.Col("Cid"),
			tablewriter.Col("State"),
		)
		for _, shard := range shards {
			tw.Write(map[string]interface{}{
				"OrderId": shard.OrderId,
				"Cid":     shard.Cid,
				"State":   shard.State,
			})
		}
		return tw.Flush(os.Stdout)
	},
}
