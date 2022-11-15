package main

// TODO:
// * init should store node address locally.
// later cmd(join, quit) should call node process api to get node address if accountAddress not provided.

import (
	"fmt"
	"sao-storage-node/api"
	"sao-storage-node/build"
	cliutil "sao-storage-node/cmd"
	"sao-storage-node/node"
	"sao-storage-node/node/repo"

	"github.com/common-nighthawk/go-figure"
	"github.com/gbrlsnchs/jwt/v3"

	"github.com/ipfs/go-datastore"
	"github.com/multiformats/go-multiaddr"

	"os"
	"sao-storage-node/chain"

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
	_ = logging.SetLogLevel("chain", "INFO")
	_ = logging.SetLogLevel("gateway", "INFO")
	_ = logging.SetLogLevel("storage", "INFO")
	_ = logging.SetLogLevel("transport", "INFO")
	_ = logging.SetLogLevel("store", "INFO")
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
			updateCmd,
			quitCmd,
			runCmd,
			authCmd,
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
			Name:  "creator",
			Usage: "node's account name",
		},
		&cli.StringFlag{
			Name:  "multiaddr",
			Usage: "nodes' multiaddr",
			Value: "/ip4/127.0.0.1/tcp/4001/",
		},
		&cli.StringFlag{
			Name:     "chainAddress",
			Value:    "http://localhost:26657",
			Required: false,
		},
	},
	Action: func(cctx *cli.Context) error {
		log.Info("Checking if repo exists")
		ctx := cctx.Context
		// TODO: validate input
		repoPath := cctx.String(FlagStorageRepo)
		creator := cctx.String("creator")
		ma, err := multiaddr.NewMultiaddr(cctx.String("multiaddr"))
		if err != nil {
			return xerrors.Errorf("invalid --multiaddr: %w", err)
		}

		chainAddress := cctx.String("chainAddress")

		r, err := initRepo(repoPath)
		if err != nil {
			return xerrors.Errorf("init repo: %w", err)
		}

		// init metadata datastore
		mds, err := r.Datastore(ctx, "/metadata")
		if err != nil {
			return err
		}
		if err := mds.Put(ctx, datastore.NewKey("node-address"), []byte(creator)); err != nil {
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

		chain, err := chain.NewChainSvc(ctx, "cosmos", chainAddress, "/websocket")
		if err != nil {
			return xerrors.Errorf("new cosmos chain: %w", err)
		}
		if tx, err := chain.Login(ctx, creator, ma, peerid); err != nil {
			// TODO: clear dir
			return err
		} else {
			fmt.Println(tx)
		}

		return nil
	},
}

func initRepo(repoPath string) (*repo.Repo, error) {
	// init base dir
	r, err := repo.NewRepo(repoPath)
	if err != nil {
		return nil, err
	}

	ok, err := r.Exists()
	if err != nil {
		return nil, err
	}
	if ok {
		return nil, xerrors.Errorf("repo at '%s' is already initialized", repoPath)
	}

	log.Info("Initializing repo")
	if err = r.Init(); err != nil {
		return nil, err
	}
	return r, nil
}

var updateCmd = &cli.Command{
	Name:  "reset",
	Usage: "update peer information.",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "creator",
			Usage: "node's account name",
		},
		&cli.StringSliceFlag{
			Name:     "multiaddrs",
			Usage:    "multiaddrs",
			Required: true,
		},
		&cli.StringFlag{
			Name:     "chainAddress",
			Value:    "http://localhost:26657",
			Required: false,
		},
	},
	Action: func(cctx *cli.Context) error {
		ctx := cctx.Context
		// TODO: validate input
		creator := cctx.String("creator")
		chainAddress := cctx.String("chainAddress")

		peerInfo := ""
		multiaddrs := cctx.StringSlice("multiaddrs")
		if len(multiaddrs) < 1 {
			return xerrors.Errorf("invalid --multiaddrs: cannot be empty")
		}

		for _, maddr := range multiaddrs {
			ma, err := multiaddr.NewMultiaddr(maddr)
			if err != nil {
				return xerrors.Errorf("invalid --multiaddrs: %w", err)
			}
			if len(peerInfo) > 0 {
				peerInfo = peerInfo + ","
			}
			peerInfo = peerInfo + ma.String()
		}

		chain, err := chain.NewChainSvc(ctx, "cosmos", chainAddress, "/websocket")
		if err != nil {
			return xerrors.Errorf("new cosmos chain: %w", err)
		}

		tx, err := chain.Reset(ctx, creator, peerInfo)
		if err != nil {
			return err
		}
		fmt.Println(tx)

		return nil
	},
}

var quitCmd = &cli.Command{
	Name:  "quit",
	Usage: "node quit sao network",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "creator",
			Usage: "node's account name",
		},
		&cli.StringFlag{
			Name:     "chainAddress",
			Value:    "http://localhost:26657",
			Required: false,
		},
	},
	Action: func(cctx *cli.Context) error {
		ctx := cctx.Context
		// TODO: validate input
		creator := cctx.String("creator")
		chainAddress := cctx.String("chainAddress")

		chain, err := chain.NewChainSvc(ctx, "cosmos", chainAddress, "/websocket")
		if err != nil {
			return xerrors.Errorf("new cosmos chain: %w", err)
		}
		if tx, err := chain.Logout(ctx, creator); err != nil {
			return err
		} else {
			fmt.Println(tx)
		}

		return nil
	},
}

var runCmd = &cli.Command{
	Name: "run",
	Action: func(cctx *cli.Context) error {
		myFigure := figure.NewFigure("Sao Network", "", true)
		myFigure.Print()

		// there is no place to trigger shutdown signal now. may add somewhere later.
		shutdownChan := make(chan struct{})
		ctx := cctx.Context

		repo, err := prepareRepo(cctx)
		if err != nil {
			return err
		}

		snode, err := node.NewNode(ctx, repo)
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

var authCmd = &cli.Command{
	Name:  "api-token-gen",
	Usage: "Generate API tokens",
	Action: func(cctx *cli.Context) error {
		repo, err := prepareRepo(cctx)
		if err != nil {
			return err
		}

		key, err := repo.GetKeyBytes()
		if err != nil {
			return err
		}

		rb, err := jwt.Sign(&node.JwtPayload{Allow: api.AllPermissions[:2]}, jwt.NewHS256(key))
		if err != nil {
			return err
		}
		fmt.Println("Read permission token  : " + string(rb))

		wb, err := jwt.Sign(&node.JwtPayload{Allow: api.AllPermissions[:3]}, jwt.NewHS256(key))
		if err != nil {
			return err
		}
		fmt.Println("Write permission token : " + string(wb))

		ab, err := jwt.Sign(&node.JwtPayload{Allow: api.AllPermissions[:4]}, jwt.NewHS256(key))
		if err != nil {
			return err
		}
		fmt.Println("Admin permission token : " + string(ab))

		return nil
	},
}

func prepareRepo(cctx *cli.Context) (*repo.Repo, error) {
	repoPath := cctx.String(FlagStorageRepo)
	repo, err := repo.NewRepo(repoPath)
	if err != nil {
		return nil, err
	}

	ok, err := repo.Exists()
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, xerrors.Errorf("repo at '%s' is not initialized, run 'snode init' to set it up", repoPath)
	}
	return repo, nil
}
