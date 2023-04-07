package main

// TODO:
// * init should store node address locally.
// later cmd(join, quit) should call node process api to get node address if accountAddress not provided.

import (
	"bufio"
	"encoding/json"
	"fmt"
	"sao-node/api"
	"sao-node/build"
	cliutil "sao-node/cmd"
	"sao-node/cmd/account"
	"sao-node/node"
	"sao-node/node/config"
	"sao-node/node/repo"
	"sao-node/types"
	"strings"

	"cosmossdk.io/math"
	"github.com/common-nighthawk/go-figure"
	"github.com/fatih/color"
	"github.com/filecoin-project/lotus/lib/tablewriter"
	"github.com/gbrlsnchs/jwt/v3"
	"golang.org/x/xerrors"

	"github.com/ipfs/go-datastore"
	"github.com/multiformats/go-multiaddr"

	"os"
	"sao-node/chain"

	manet "github.com/multiformats/go-multiaddr/net"

	nodetypes "github.com/SaoNetwork/sao/x/node/types"
	logging "github.com/ipfs/go-log/v2"
	"github.com/libp2p/go-libp2p/core/peer"
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
			initCmd,
			joinCmd,
			cleanCmd,
			updateCmd,
			peersCmd,
			runCmd,
			authCmd,
			migrateCmd,
			infoCmd,
			claimCmd,
			jobsCmd,
			account.AccountCmd,
			cliutil.GenerateDocCmd,
		},
	}
	app.Setup()

	if err := app.Run(os.Args); err != nil {
		os.Stderr.WriteString("Error: " + err.Error() + "\n")
		os.Exit(1)
	}
}

var jobsCmd = &cli.Command{
	Name: "job",
	Subcommands: []*cli.Command{
		ordersCmd,
		shardsCmd,
		migrationsCmd,
	},
}

var initCmd = &cli.Command{
	Name:  "init",
	Usage: "initialize a sao network node",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:     "creator",
			Usage:    "node's account on sao chain",
			Required: true,
		},
		&cli.StringFlag{
			Name:     "multiaddr",
			Usage:    "nodes' multiaddr",
			Value:    "/ip4/127.0.0.1/tcp/5153/",
			Required: false,
		},
	},
	Action: func(cctx *cli.Context) error {
		ctx := cctx.Context

		chainAddress := cliutil.ChainAddress
		if chainAddress == "" {
			return types.Wrapf(types.ErrInvalidParameters, "must provide --chain-address")
		}

		repoPath := cctx.String(FlagStorageRepo)
		creator := cctx.String("creator")

		r, err := initRepo(repoPath, chainAddress)
		if err != nil {
			return err
		}

		c, err := r.Config()
		if err != nil {
			return types.Wrapf(types.ErrReadConfigFailed, "invalid config for repo, got: %T", c)
		}

		// init metadata datastore
		mds, err := r.Datastore(ctx, "/metadata")
		if err != nil {
			return types.Wrap(types.ErrOpenDataStoreFailed, err)
		}
		if err := mds.Put(ctx, datastore.NewKey("node-address"), []byte(creator)); err != nil {
			return types.Wrap(types.ErrGetFailed, err)
		}

		log.Info("initialize libp2p identity")

		chain, err := chain.NewChainSvc(ctx, chainAddress, "/websocket", cliutil.KeyringHome)
		if err != nil {
			return err
		}

		for {
			fmt.Printf("Please make sure there is enough SAO tokens in the account %s. Confirm with 'yes' :", creator)

			reader := bufio.NewReader(os.Stdin)
			indata, err := reader.ReadBytes('\n')
			if err != nil {
				return types.Wrap(types.ErrInvalidParameters, err)
			}
			if strings.ToLower(strings.Replace(string(indata), "\n", "", -1)) != "yes" {
				continue
			}

			coins, err := chain.GetBalance(ctx, creator)
			if err != nil {
				fmt.Printf("%v", err)
				continue
			} else {
				if coins.AmountOf("sao").LT(math.NewInt(1000)) {
					continue
				} else {
					break
				}
			}

		}

		if tx, err := chain.Create(ctx, creator); err != nil {
			// TODO: clear dir
			return err
		} else {
			fmt.Println(tx)
		}

		return nil
	},
}

func initRepo(repoPath string, chainAddress string) (*repo.Repo, error) {
	// init base dir
	r, err := repo.NewRepo(repoPath)
	if err != nil {
		return nil, err
	}

	ok, err := r.Exists()
	if err != nil {
		return nil, types.Wrap(types.ErrOpenRepoFailed, err)
	}

	if ok {
		return nil, types.Wrapf(types.ErrInitRepoFailed, "repo at '%s' is already initialized", repoPath)
	}

	log.Info("Initializing repo")
	if err = r.Init(chainAddress); err != nil {
		return nil, err
	}
	return r, nil
}

var joinCmd = &cli.Command{
	Name:  "join",
	Usage: "join sao network",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:     "creator",
			Usage:    "node's account on sao chain",
			Required: true,
		},
	},
	Action: func(cctx *cli.Context) error {
		ctx := cctx.Context

		chainAddress, err := cliutil.GetChainAddress(cctx, cctx.String("repo"), cctx.App.Name)
		if err != nil {
			log.Warn(err)
		}
		creator := cctx.String("creator")

		chain, err := chain.NewChainSvc(ctx, chainAddress, "/websocket", cliutil.KeyringHome)
		if err != nil {
			return err
		}

		repo, err := prepareRepo(cctx)
		if err != nil {
			return err
		}
		c, err := repo.Config()
		if err != nil {
			return types.Wrapf(types.ErrReadConfigFailed, "invalid config for repo, got: %T", c)
		}

		cfg, ok := c.(*config.Node)
		if !ok {
			return types.Wrapf(types.ErrDecodeConfigFailed, "invalid config for repo, got: %T", c)
		}
		var status = node.NODE_STATUS_ONLINE
		if cfg.Module.GatewayEnable {
			status = status | node.NODE_STATUS_SERVE_GATEWAY
		}
		if cfg.Module.StorageEnable {
			status = status | node.NODE_STATUS_SERVE_STORAGE
			if cctx.Bool("accept-order") {
				status = status | node.NODE_STATUS_ACCEPT_ORDER
			} else if cfg.Storage.AcceptOrder {
				status = status | node.NODE_STATUS_ACCEPT_ORDER
			}
		}

		tx, err := chain.Create(ctx, creator)
		if err != nil {
			return err
		} else {
			fmt.Println(tx)
		}

		// update metadata datastore
		mds, err := repo.Datastore(ctx, "/metadata")
		if err != nil {
			return types.Wrap(types.ErrOpenDataStoreFailed, err)
		}
		if err := mds.Put(ctx, datastore.NewKey("node-address"), []byte(creator)); err != nil {
			return types.Wrap(types.ErrGetFailed, err)
		}

		return nil
	},
}

var cleanCmd = &cli.Command{
	Name:  "clean",
	Usage: "clean up the local datastore",
	Action: func(cctx *cli.Context) error {
		ctx := cctx.Context

		console := color.New(color.FgRed, color.Bold)
		console.Println("!!!BE CAREFULL!!!")
		console.Print("It'll remove all the configurations in the local datastore and you have to init a new storage node. Confirm with 'yes' :")
		reader := bufio.NewReader(os.Stdin)
		indata, err := reader.ReadBytes('\n')
		if err != nil {
			return types.Wrap(types.ErrInvalidParameters, err)
		}
		if strings.ToLower(strings.Replace(string(indata), "\n", "", -1)) == "yes" {
			repo, err := prepareRepo(cctx)
			if err != nil {
				return err
			}

			mds, err := repo.Datastore(ctx, "/metadata")
			if err != nil {
				return types.Wrap(types.ErrOpenDataStoreFailed, err)
			}
			mds.Delete(ctx, datastore.NewKey("node-address"))
			console.Println("Node address information has been deleted!")

			tds, err := repo.Datastore(ctx, "/transport")
			if err != nil {
				return types.Wrap(types.ErrOpenDataStoreFailed, err)
			}
			tds.Delete(ctx, datastore.NewKey(fmt.Sprintf(types.PEER_INFO_PREFIX)))
			console.Println("Peer information has been deleted!")
		}

		return nil
	},
}

var updateCmd = &cli.Command{
	Name:  "update",
	Usage: "update node information",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "creator",
			Usage: "node's account on sao chain",
		},
		&cli.StringSliceFlag{
			Name:     "multiaddrs",
			Usage:    "node's multiaddrs",
			Required: false,
		},
		&cli.BoolFlag{
			Name:     "accept-order",
			Usage:    "whether this node can accept shard as a storage node",
			Value:    true,
			Required: false,
		},
	},
	Action: func(cctx *cli.Context) error {
		ctx := cctx.Context
		// TODO: validate input
		creator := cctx.String("creator")

		var peerInfo = ""
		if cctx.IsSet("multiaddrs") {
			multiaddrs := cctx.StringSlice("multiaddrs")
			if len(multiaddrs) < 1 {
				return types.Wrapf(types.ErrInvalidParameters, "invalid --multiaddrs: cannot be empty")
			}

			for _, maddr := range multiaddrs {
				ma, err := multiaddr.NewMultiaddr(maddr)
				if err != nil {
					return types.Wrapf(types.ErrInvalidParameters, "invalid --multiaddrs: %v", err)
				}
				if len(peerInfo) > 0 {
					peerInfo = peerInfo + ","
				}
				peerInfo = peerInfo + ma.String()
			}
		}

		r, err := prepareRepo(cctx)
		if err != nil {
			return err
		}

		c, err := r.Config()
		if err != nil {
			return types.Wrapf(types.ErrReadConfigFailed, "invalid config for repo, got: %T", c)
		}

		cfg, ok := c.(*config.Node)
		if !ok {
			return types.Wrapf(types.ErrDecodeConfigFailed, "invalid config for repo, got: %T", c)
		}

		chainAddress, err := cliutil.GetChainAddress(cctx, cctx.String("repo"), cctx.App.Name)
		if err != nil {
			log.Warn(err)
		}

		chain, err := chain.NewChainSvc(ctx, chainAddress, "/websocket", cliutil.KeyringHome)
		if err != nil {
			return err
		}

		var status = node.NODE_STATUS_ONLINE
		if cfg.Module.GatewayEnable {
			status = status | node.NODE_STATUS_SERVE_GATEWAY
		}
		if cfg.Module.StorageEnable {
			status = status | node.NODE_STATUS_SERVE_STORAGE
			if cctx.Bool("accept-order") {
				status = status | node.NODE_STATUS_ACCEPT_ORDER
			} else if cfg.Storage.AcceptOrder {
				status = status | node.NODE_STATUS_ACCEPT_ORDER
			}
		}

		tx, err := chain.Reset(ctx, creator, peerInfo, status)
		if err != nil {
			return err
		}
		fmt.Println(tx)

		return nil
	},
}

var peersCmd = &cli.Command{
	Name:  "peers",
	Usage: "show p2p peer list",
	Action: func(cctx *cli.Context) error {
		ctx := cctx.Context

		apiClient, closer, err := cliutil.GetNodeApi(cctx, cctx.String(FlagStorageRepo), NodeApi, cliutil.ApiToken)
		if err != nil {
			return err
		}
		defer closer()

		peers, err := apiClient.GetNetPeers(ctx)
		if err != nil {
			return err
		}

		seen := make(map[peer.ID]struct{})

		console := color.New(color.FgMagenta, color.Bold)

		if len(peers) == 0 {
			console.Println(" no peer connected...")
		}

		for _, peer := range peers {
			_, dup := seen[peer.ID]
			if dup {
				continue
			}
			seen[peer.ID] = struct{}{}

			if err != nil {
				console.Printf(" error getting peer info: %s\r\n", err)
			} else {
				bytes, err := json.Marshal(&peer)
				if err != nil {
					console.Printf(" error marshalling peer info: %s\r\n", err)
				} else {
					console.Println(string(bytes))
				}
			}
		}

		return nil
	},
}

var runCmd = &cli.Command{
	Name:  "run",
	Usage: "start node",
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

		snode, err := node.NewNode(ctx, repo, cliutil.KeyringHome)
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

var infoCmd = &cli.Command{
	Name:  "info",
	Usage: "show node information",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:     "creator",
			Usage:    "node's account on sao chain",
			Required: false,
		},
	},
	Action: func(cctx *cli.Context) error {
		ctx := cctx.Context

		chainAddress, err := cliutil.GetChainAddress(cctx, cctx.String("repo"), cctx.App.Name)
		if err != nil {
			log.Warn(err)
		}

		chain, err := chain.NewChainSvc(ctx, chainAddress, "/websocket", cliutil.KeyringHome)
		if err != nil {
			return err
		}

		creator := cctx.String("creator")
		if creator == "" {
			apiClient, closer, err := cliutil.GetNodeApi(cctx, cctx.String(FlagStorageRepo), NodeApi, cliutil.ApiToken)
			if err != nil {
				return types.Wrap(types.ErrCreateClientFailed, err)
			}
			defer closer()

			creator, err = apiClient.GetNodeAddress(ctx)
			if err != nil {
				return err
			}
		}
		chain.ShowBalance(ctx, creator)
		chain.ShowNodeInfo(ctx, creator)

		return nil
	},
}

var migrateCmd = &cli.Command{
	Name: "migrate",
	Action: func(cctx *cli.Context) error {
		ctx := cctx.Context
		if cctx.Args().Len() != 1 {
			return xerrors.Errorf("missing data ids parameter")
		}
		dataIds := strings.Split(cctx.Args().First(), ",")

		apiClient, closer, err := cliutil.GetNodeApi(cctx, cctx.String(FlagStorageRepo), NodeApi, cliutil.ApiToken)
		if err != nil {
			return err
		}
		defer closer()

		resp, err := apiClient.ModelMigrate(ctx, dataIds)
		if err != nil {
			return err
		}
		fmt.Println(resp.TxHash)
		tw := tablewriter.New(
			tablewriter.Col("DataId"),
			tablewriter.Col("Result"),
		)
		for k, v := range resp.Results {
			tw.Write(map[string]interface{}{
				"DataId": k,
				"Result": v,
			})

		}
		return tw.Flush(os.Stdout)
	},
}

var claimCmd = &cli.Command{
	Name:  "claim",
	Usage: "claim sao network storage reward",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:     "creator",
			Usage:    "node's account on sao chain",
			Required: false,
		},
	},
	Action: func(cctx *cli.Context) error {
		ctx := cctx.Context

		creator := cctx.String("creator")
		if creator == "" {
			apiClient, closer, err := cliutil.GetNodeApi(cctx, cctx.String(FlagStorageRepo), NodeApi, cliutil.ApiToken)
			if err != nil {
				return types.Wrap(types.ErrCreateClientFailed, err)
			}
			defer closer()

			creator, err = apiClient.GetNodeAddress(ctx)
			if err != nil {
				return err
			}
		}

		chainAddress, err := cliutil.GetChainAddress(cctx, cctx.String("repo"), cctx.App.Name)
		if err != nil {
			log.Warn(err)
		}

		chain, err := chain.NewChainSvc(ctx, chainAddress, "/websocket", cliutil.KeyringHome)
		if err != nil {
			return err
		}

		if tx, err := chain.ClaimReward(ctx, creator); err != nil {
			return err
		} else {
			fmt.Println(tx)
		}

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

		console := color.New(color.FgMagenta, color.Bold)

		rb, err := jwt.Sign(&node.JwtPayload{Allow: api.AllPermissions[:2]}, jwt.NewHS256(key))
		if err != nil {
			return types.Wrap(types.ErrSignedFailed, err)
		}
		fmt.Print(" Read permission token   : ")
		console.Println(string(rb))

		wb, err := jwt.Sign(&node.JwtPayload{Allow: api.AllPermissions[:3]}, jwt.NewHS256(key))
		if err != nil {
			return types.Wrap(types.ErrSignedFailed, err)
		}
		fmt.Print(" Write permission token  : ")
		console.Println(string(wb))

		ab, err := jwt.Sign(&node.JwtPayload{Allow: api.AllPermissions[:4]}, jwt.NewHS256(key))
		if err != nil {
			return types.Wrap(types.ErrSignedFailed, err)
		}
		fmt.Print(" Admin permission token  : ")
		console.Println(string(ab))

		return nil
	},
}

func prepareRepo(cctx *cli.Context) (*repo.Repo, error) {
	return repo.PrepareRepo(cctx.String(FlagStorageRepo))
}
