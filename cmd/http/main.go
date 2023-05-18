package main

// TODO:
// * init should store node address locally.
// later cmd(join, quit) should call node process api to get node address if accountAddress not provided.

import (
	"path"
	"sao-node/build"
	cliutil "sao-node/cmd"
	"sao-node/node"
	"sao-node/node/config"
	"sao-node/node/gateway"
	"sao-node/node/repo"
	"sao-node/types"

	"github.com/common-nighthawk/go-figure"

	"os"

	logging "github.com/ipfs/go-log/v2"
	"github.com/urfave/cli/v2"
)

var log = logging.Logger("node")

const (
	FlagStorageRepo        = "repo"
	FlagStorageDefaultRepo = "~/.sao-node"
)

var NodeApi string
var FlagNodeApi = &cli.StringFlag{
	Name:        "node",
	Usage:       "node connection",
	EnvVars:     []string{"SAO_NODE_API"},
	Required:    false,
	Destination: &NodeApi,
}

var FlagRepo = &cli.StringFlag{
	Name:    FlagStorageRepo,
	Usage:   "repo directory for sao storage node",
	EnvVars: []string{"SAO_NODE_PATH"},
	Value:   FlagStorageDefaultRepo,
}

func before(_ *cli.Context) error {
	_ = logging.SetLogLevel("cache", "INFO")
	_ = logging.SetLogLevel("model", "INFO")
	_ = logging.SetLogLevel("node", "INFO")
	_ = logging.SetLogLevel("rpc", "INFO")
	_ = logging.SetLogLevel("chain", "INFO")
	_ = logging.SetLogLevel("gateway", "INFO")
	_ = logging.SetLogLevel("storage", "INFO")
	_ = logging.SetLogLevel("transport", "INFO")
	_ = logging.SetLogLevel("store", "INFO")
	_ = logging.SetLogLevel("indexer", "INFO")
	_ = logging.SetLogLevel("graphql", "INFO")
	if cliutil.IsVeryVerbose {
		_ = logging.SetLogLevel("cache", "DEBUG")
		_ = logging.SetLogLevel("model", "DEBUG")
		_ = logging.SetLogLevel("node", "DEBUG")
		_ = logging.SetLogLevel("rpc", "DEBUG")
		_ = logging.SetLogLevel("chain", "DEBUG")
		_ = logging.SetLogLevel("gateway", "DEBUG")
		_ = logging.SetLogLevel("storage", "DEBUG")
		_ = logging.SetLogLevel("transport", "DEBUG")
		_ = logging.SetLogLevel("store", "DEBUG")
		_ = logging.SetLogLevel("indexer", "DEBUG")
		_ = logging.SetLogLevel("graphql", "DEBUG")
	}

	return nil
}

func main() {
	app := &cli.App{
		Name:                 cliutil.APP_NAME_NODE,
		Usage:                "Command line for sao network node",
		EnableBashCompletion: true,
		Version:              build.UserVersion(),
		Before:               before,
		Flags: []cli.Flag{
			FlagRepo,
			cliutil.FlagChainAddress,
			cliutil.FlagVeryVerbose,
			cliutil.FlagKeyringHome,
			FlagNodeApi,
			cliutil.FlagToken,
		},
		Commands: []*cli.Command{
			runCmd,
		},
	}
	app.Setup()

	if err := app.Run(os.Args); err != nil {
		os.Stderr.WriteString("Error: " + err.Error() + "\n")
		os.Exit(1)
	}
}

var runCmd = &cli.Command{
	Name:  "run",
	Usage: "start node",
	Action: func(cctx *cli.Context) error {
		myFigure := figure.NewFigure("Sao Network", "", true)
		myFigure.Print()

		// there is no place to trigger shutdown signal now. may add somewhere later.
		shutdownChan := make(chan struct{})

		repo, err := prepareRepo(cctx)
		if err != nil {
			return err
		}
		serverPath := path.Join(repo.Path, "http-files")

		c, err := repo.Config()
		cfg, ok := c.(*config.Node)
		if !ok {
			return types.Wrapf(types.ErrDecodeConfigFailed, "invalid config for repo, got: %T", c)
		}

		hfs, err := gateway.StartHttpFileServer(serverPath, &cfg.SaoHttpFileServer, cfg, cctx)
		if err != nil {
			return err
		}

		finishCh := node.MonitorShutdown(
			shutdownChan,
			node.ShutdownHandler{Component: "storagenode", StopFunc: hfs.Stop},
		)
		<-finishCh
		return nil
	},
}

func prepareRepo(cctx *cli.Context) (*repo.Repo, error) {
	return repo.PrepareRepo(cctx.String(FlagStorageRepo))
}
