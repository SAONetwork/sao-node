package main

import (
	"os"
	apiclient "sao-node/api/client"
	cliutil "sao-node/cmd"

	"github.com/filecoin-project/lotus/lib/tablewriter"
	"github.com/urfave/cli/v2"
)

var migrationsCmd = &cli.Command{
	Name:  "migrations",
	Usage: "migration job management",
	Subcommands: []*cli.Command{
		migrateListCmd,
	},
}

var migrateListCmd = &cli.Command{
	Name:  "list",
	Usage: "List migration jobs",
	Action: func(cctx *cli.Context) error {
		ctx := cctx.Context
		gatewayApi, closer, err := apiclient.NewGatewayApi(ctx, cliutil.Gateway, "DEFAULT_TOKEN")
		if err != nil {
			return err
		}
		defer closer()

		jobs, err := gatewayApi.MigrateJobList(ctx)
		if err != nil {
			return err
		}

		tw := tablewriter.New(
			tablewriter.Col("OrderId"),
			tablewriter.Col("DataId"),
			tablewriter.Col("Cid"),
			tablewriter.Col("To"),
			tablewriter.Col("State"),
		)
		for _, job := range jobs {
			tw.Write(map[string]interface{}{
				"OrderId": job.OrderId,
				"DataId":  job.DataId,
				"Cid":     job.Cid,
				"To":      job.ToProvider,
				"State":   job.State,
			})
		}
		return tw.Flush(os.Stdout)
	},
}
