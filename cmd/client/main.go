package main

// TODO:
// * how to generate cid from scratch
// * guic transfer data

import (
	"os"
	"sao-storage-node/build"
	"sao-storage-node/client"
	cliutil "sao-storage-node/cmd"
	"sao-storage-node/cmd/account"

	logging "github.com/ipfs/go-log/v2"
	"github.com/urfave/cli/v2"
)

const (
	DEFAULT_DURATION = 365
	DEFAULT_REPLICA  = 1

	FlagClientRepo = "repo"
	FlagGateway    = "gateway"
)

var FlagRepo = &cli.StringFlag{
	Name:     FlagClientRepo,
	Usage:    "repo directory for sao client",
	Required: false,
	EnvVars:  []string{"SAO_CLIENT_PATH"},
	Value:    "~/.sao-cli",
}

var gateway = &cli.StringFlag{
	Name:     FlagGateway,
	EnvVars:  []string{"SAO_GATEWAY_API"},
	Required: false,
}

var platform = &cli.StringFlag{
	Name:     "platform",
	Usage:    "platform to manage the data model",
	Required: false,
}

func getSaoClient(cctx *cli.Context) (*client.SaoClient, error) {
	return client.NewSaoClient(cctx.Context, cctx.String(FlagClientRepo), cctx.String(FlagGateway))
}

func before(cctx *cli.Context) error {
	// by default, do not print any log for client.
	_ = logging.SetLogLevel("saoclient", "TRACE")
	_ = logging.SetLogLevel("chain", "TRACE")
	_ = logging.SetLogLevel("transport-client", "TRACE")

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
			FlagRepo,
			gateway,
			platform,
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
