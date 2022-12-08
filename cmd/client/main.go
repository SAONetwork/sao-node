package main

// TODO:
// * how to generate cid from scratch
// * guic transfer data

import (
	"os"
	"sao-storage-node/build"
	cliutil "sao-storage-node/cmd"
	"sao-storage-node/cmd/account"

	logging "github.com/ipfs/go-log/v2"
	"github.com/urfave/cli/v2"
)

const (
	DEFAULT_DURATION = 365
	DEFAULT_REPLICA  = 1
)

var gateway = &cli.StringFlag{
	Name:     "gateway",
	EnvVars:  []string{"SAO_GATEWAY_API"},
	Required: false,
}

var platform = &cli.StringFlag{
	Name:     "platform",
	Usage:    "platform to manage the data model",
	Required: false,
}

var secret = &cli.StringFlag{
	Name:     "secret",
	Usage:    "client secret",
	Required: false,
}

func before(cctx *cli.Context) error {
	_ = logging.SetLogLevel("saoclient", "INFO")
	_ = logging.SetLogLevel("chain", "INFO")
	_ = logging.SetLogLevel("transport-client", "INFO")

	if cliutil.IsVeryVerbose {
		_ = logging.SetLogLevel("saoclient", "DEBUG")
		_ = logging.SetLogLevel("chain", "DEBUG")
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
			cliutil.FlagChainAddress,
			gateway,
			platform,
			secret,
			cliutil.FlagNetType,
			cliutil.FlagVeryVerbose,
		},
		Commands: []*cli.Command{
			account.AccountCmd,
			modelCmd,
			fileCmd,
			didCmd,
		},
	}
	app.Setup()

	if err := app.Run(os.Args); err != nil {
		os.Stderr.WriteString("Error: " + err.Error() + "\n")
		os.Exit(1)
	}
}
