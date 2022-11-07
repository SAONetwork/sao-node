package main

// TODO:
// * how to generate cid from scratch
// * guic transfer data

import (
	"os"
	"sao-storage-node/build"
	cliutil "sao-storage-node/cmd"

	logging "github.com/ipfs/go-log/v2"
	"github.com/urfave/cli/v2"
)

var log = logging.Logger("saoclient")

const (
	DEFAULT_DURATION = 365
	DEFAULT_REPLICA  = 1
)

func before(cctx *cli.Context) error {
	_ = logging.SetLogLevel("saoclient", "INFO")
	_ = logging.SetLogLevel("transport-client", "INFO")

	if cliutil.IsVeryVerbose {
		_ = logging.SetLogLevel("saoclient", "DEBUG")
		_ = logging.SetLogLevel("transport-client", "DEBUG")
	}

	return nil
}

func main() {
	app := &cli.App{
		Name:                 "saoclient",
		Usage:                "cli client for network client",
		EnableBashCompletion: true,
		Version:              build.UserVersion(),
		Before:               before,
		Flags: []cli.Flag{
			cliutil.FlagVeryVerbose,
		},
		Commands: []*cli.Command{
			testCmd,
			modelCmd,
			fileCmd,
		},
	}
	app.Setup()

	if err := app.Run(os.Args); err != nil {
		os.Stderr.WriteString("Error: " + err.Error() + "\n")
		os.Exit(1)
	}
}
