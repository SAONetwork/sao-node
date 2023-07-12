package main

import (
	"fmt"
	"os"

	cliutil "github.com/SaoNetwork/sao-node/cmd"

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
		apiClient, closer, err := cliutil.GetNodeApi(cctx, cctx.String(FlagStorageRepo), NodeApi, cliutil.ApiToken)
		if err != nil {
			return err
		}
		defer closer()

		jobs, err := apiClient.MigrateJobList(ctx)
		if err != nil {
			return err
		}

		if len(jobs) > 0 {
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
					"State":   job.State.String(),
				})
			}
			return tw.Flush(os.Stdout)
		} else {
			fmt.Println("No migration jobs.")
			return nil
		}
	},
}
