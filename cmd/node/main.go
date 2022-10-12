package main

// TODO:
// * init should store node address locally.
// later cmd(join, quit) should call node process api to get node address if accountAddress not provided.

import (
	"fmt"
	"os"
	"sao-storage-node/build"
	cliutil "sao-storage-node/cmd"
	"sao-storage-node/node"
	"sao-storage-node/node/repo"

	"github.com/SaoNetwork/sao/x/node/types"
	"github.com/ignite/cli/ignite/pkg/cosmosclient"
	logging "github.com/ipfs/go-log/v2"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/urfave/cli/v2"
	"golang.org/x/xerrors"
)

var log = logging.Logger("node")

const (
	FlagStorageRepo        = "repo"
	FlagStorageDefaultRepo = "~/.sao-storage-node"
)

var FlagRepo = &cli.StringFlag{
	Name:    FlagStorageRepo,
	Usage:   "repo directory for sao storage node",
	EnvVars: []string{"SAO_NODE_PATH"},
	Value:   FlagStorageDefaultRepo,
}

func before(cctx *cli.Context) error {
	_ = logging.SetLogLevel("cache", "INFO")
	_ = logging.SetLogLevel("model", "INFO")
	_ = logging.SetLogLevel("node", "INFO")
	_ = logging.SetLogLevel("rpc", "INFO")
	_ = logging.SetLogLevel("transport", "INFO")
	if cliutil.IsVeryVerbose {
		_ = logging.SetLogLevel("cache", "DEBUG")
		_ = logging.SetLogLevel("model", "DEBUG")
		_ = logging.SetLogLevel("node", "DEBUG")
		_ = logging.SetLogLevel("rpc", "DEBUG")
		_ = logging.SetLogLevel("transport", "DEBUG")
	}

	return nil
}

func main() {
	app := &cli.App{
		Name:                 "snode",
		EnableBashCompletion: true,
		Version:              build.UserVersion(),
		Before:               before,
		Flags: []cli.Flag{
			FlagRepo,
			cliutil.FlagVeryVerbose,
		},
		Commands: []*cli.Command{
			initCmd,
			joinCmd,
			updateCmd,
			quitCmd,
			runCmd,
		},
	}
	app.Setup()

	if err := app.Run(os.Args); err != nil {
		os.Stderr.WriteString("Error: " + err.Error() + "\n")
		os.Exit(1)
	}
}

var initCmd = &cli.Command{
	Name: "init",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "accountName",
			Usage: "node's account name",
		},
	},
	Action: func(cctx *cli.Context) error {
		log.Info("Checking if repo exists")

		repoPath := cctx.String(FlagStorageRepo)
		r, err := repo.NewRepo(repoPath)
		if err != nil {
			return err
		}

		ok, err := r.Exists()
		if err != nil {
			return err
		}
		if ok {
			return xerrors.Errorf("repo at '%s' is already initialized", cctx.String(FlagStorageRepo))
		}

		log.Info("Initializing repo")
		if err := r.Init(); err != nil {
			return err
		}

		log.Info("initialize libp2p identity")
		p2pSk, err := r.GeneratePeerId()
		if err != nil {
			return xerrors.Errorf("make host key: %w", err)
		}

		peerid, err := peer.IDFromPrivateKey(p2pSk)
		if err != nil {
			return xerrors.Errorf("peer ID from private key: %w", err)
		}

		addressPrefix := "cosmos"
		cosmos, err := cosmosclient.New(cctx.Context, cosmosclient.WithAddressPrefix(addressPrefix))
		if err != nil {
			return err
		}

		account, err := cosmos.Account(cctx.String("accountName"))
		if err != nil {
			return err
		}

		addr, err := account.Address(addressPrefix)
		if err != nil {
			return err
		}

		// TODO: /ip4/127.0.0.1/tcp/4001 should be read from config.toml file.
		multiaddress := "/ip4/127.0.0.1/tcp/4001"
		msg := &types.MsgLogin{
			Creator: addr,
			Peer:    fmt.Sprintf("%v/p2p/%v", multiaddress, peerid),
		}

		// TODO: recheck - seems BroadcastTx will return after confirmed on chain.
		txResp, err := cosmos.BroadcastTx(account, msg)
		if err != nil {
			return err
		}
		if txResp.TxResponse.Code != 0 {
			return xerrors.Errorf("MsgLogin transaction %v failed: code=%d", txResp.TxResponse.TxHash, txResp.TxResponse.Code)
		} else {
			log.Infof("MsgLogin transaction %v succeed.", txResp.TxResponse.TxHash)
		}

		return nil
	},
}

var joinCmd = &cli.Command{
	Name:  "join",
	Usage: "if a node quits on chain, join cmd can allow it to re-join the network again.",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "accountName",
			Usage: "node's account name",
		},
	},
	Action: func(cctx *cli.Context) error {
		// TODO:
		return nil
	},
}

var updateCmd = &cli.Command{
	Name:  "reset",
	Usage: "update peer information.",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "accountName",
			Usage: "node's account name",
		},
		&cli.StringFlag{
			Name:  "peer",
			Usage: "peer including multiaddr and peer id",
		},
	},
	Action: func(cctx *cli.Context) error {
		addressPrefix := "cosmos"
		cosmos, err := cosmosclient.New(cctx.Context, cosmosclient.WithAddressPrefix(addressPrefix))
		if err != nil {
			return err
		}

		account, err := cosmos.Account(cctx.String("accountName"))
		if err != nil {
			return err
		}

		addr, err := account.Address(addressPrefix)
		if err != nil {
			return err
		}

		// TODO: validate peer
		peer := cctx.String("peer")
		msg := &types.MsgReset{
			Creator: addr,
			Peer:    peer,
		}
		txResp, err := cosmos.BroadcastTx(account, msg)
		if err != nil {
			return err
		}
		if txResp.TxResponse.Code != 0 {
			return xerrors.Errorf("MsgReset transaction %v failed: code=%d", txResp.TxResponse.TxHash, txResp.TxResponse.Code)
		} else {
			log.Infof("MsgReset transaction %v succeed.", txResp.TxResponse.TxHash)
		}

		return nil
	},
}

var quitCmd = &cli.Command{
	Name:  "quit",
	Usage: "node quit sao network",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "accountName",
			Usage: "node's account name",
		},
	},
	Action: func(cctx *cli.Context) error {
		addressPrefix := "cosmos"
		cosmos, err := cosmosclient.New(cctx.Context, cosmosclient.WithAddressPrefix(addressPrefix))
		if err != nil {
			return err
		}

		account, err := cosmos.Account(cctx.String("accountName"))
		if err != nil {
			return err
		}

		addr, err := account.Address(addressPrefix)
		if err != nil {
			return err
		}

		msg := &types.MsgLogout{
			Creator: addr,
		}
		txResp, err := cosmos.BroadcastTx(account, msg)
		if err != nil {
			return err
		}
		if txResp.TxResponse.Code != 0 {
			return xerrors.Errorf("MsgLogout transaction %v failed: code=%d", txResp.TxResponse.TxHash, txResp.TxResponse.Code)
		} else {
			log.Infof("MsgLogout transaction %v succeed.", txResp.TxResponse.TxHash)
		}
		return nil
	},
}

var runCmd = &cli.Command{
	Name: "run",
	Action: func(cctx *cli.Context) error {
		shutdownChan := make(chan struct{})
		ctx := cctx.Context

		repoPath := cctx.String(FlagStorageRepo)
		repo, err := repo.NewRepo(repoPath)
		if err != nil {
			return err
		}

		ok, err := repo.Exists()
		if err != nil {
			return err
		}
		if !ok {
			return xerrors.Errorf("repo at '%s' is not initialized, run 'snode init' to set it up", repoPath)
		}

		// start node.
		snode, err := node.NewStorageNode(ctx, repo)
		if err != nil {
			return err
		}
		err = snode.Start()
		if err != nil {
			return err
		}

		finishCh := node.MonitorShutdown(
			shutdownChan,
			node.ShutdownHandler{Component: "storagenode", StopFunc: snode.Stop},
		)
		<-finishCh
		return nil
	},
}
