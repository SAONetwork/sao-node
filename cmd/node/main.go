package main

// TODO:
// * init should store node address locally.
// later cmd(join, quit) should call node process api to get node address if accountAddress not provided.

import (
	"bufio"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/SaoNetwork/sao-node/api"
	"github.com/SaoNetwork/sao-node/build"
	cliutil "github.com/SaoNetwork/sao-node/cmd"
	"github.com/SaoNetwork/sao-node/cmd/account"
	"github.com/SaoNetwork/sao-node/node"
	"github.com/SaoNetwork/sao-node/node/config"
	"github.com/SaoNetwork/sao-node/node/repo"
	"github.com/SaoNetwork/sao-node/types"

	"cosmossdk.io/math"
	"github.com/common-nighthawk/go-figure"
	"github.com/fatih/color"
	"github.com/filecoin-project/lotus/lib/tablewriter"
	"github.com/gbrlsnchs/jwt/v3"
	"golang.org/x/xerrors"

	"github.com/ipfs/go-datastore"
	"github.com/multiformats/go-multiaddr"

	"os"

	"github.com/SaoNetwork/sao-node/chain"

	nodetypes "github.com/SaoNetwork/sao/x/node/types"
	saotypes "github.com/SaoNetwork/sao/x/sao/types"

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
			initCmd,
			joinCmd,
			cleanCmd,
			addStorageCmd,
			removeStorageCmd,
			updateCmd,
			peersCmd,
			runCmd,
			authCmd,
			migrateCmd,
			infoCmd,
			claimCmd,
			declareFaultsRecoverCmd,
			jobsCmd,
			initTxAddressPoolCmd,
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

var initTxAddressPoolCmd = &cli.Command{
	Name:  "init-tx-address-pool",
	Usage: "initialize tx address pool for a sao network node",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:     "creator",
			Usage:    "node's account on sao chain",
			Required: true,
		},
		&cli.UintFlag{
			Name:     "tx-pool-size",
			Usage:    "address pool size for sending message, the default value is 1",
			Value:    1,
			Required: false,
		},
		&cli.UintFlag{
			Name:     "pool-token-amount",
			Usage:    "SAO token amount reserved for address pool, the default value is 0",
			Value:    0,
			Required: false,
		},
	},
	Action: func(cctx *cli.Context) error {
		ctx := cctx.Context

		chainAddress, err := cliutil.GetChainAddress(cctx, cctx.String("repo"), cctx.App.Name)
		if err != nil {
			log.Warn(err)
		}
		creator := cctx.String("creator")
		txPoolSize := cctx.Uint("tx-pool-size")
		poolTokenAmount := cctx.Uint("pool-token-amount")

		chainSvc, err := chain.NewChainSvc(ctx, chainAddress, "/websocket", cliutil.KeyringHome)
		if err != nil {
			return err
		}

		if txPoolSize <= 0 {
			return types.Wrapf(types.ErrInvalidParameters, "tx-pool-size should greater than 0")
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

			coins, err := chainSvc.GetBalance(ctx, creator)
			if err != nil {
				fmt.Printf("%v", err)
				continue
			} else {
				if coins.AmountOf(chain.DENOM).LT(math.NewInt(int64(poolTokenAmount + 1000))) {
					continue
				} else {
					break
				}
			}

		}

		err = chain.CreateAddressPool(ctx, cliutil.KeyringHome, txPoolSize)
		if err != nil {
			return err
		}

		ap, err := chain.LoadAddressPool(ctx, cliutil.KeyringHome, txPoolSize)
		if err != nil {
			return err
		}

		if poolTokenAmount > 0 {
			for address := range ap.Addresses {
				amount := int64(poolTokenAmount / txPoolSize)
				if tx, err := chainSvc.Send(ctx, creator, address, amount); err != nil {
					// TODO: clear dir
					return err
				} else {
					fmt.Printf("Sent %d SAO from creator %s to pool address %s, txhash=%s\r", amount, creator, address, tx)
				}
			}
		}

		return nil
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
		&cli.UintFlag{
			Name:     "tx-pool-size",
			Usage:    "address pool size for sending message, the default value is 0",
			Value:    0,
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
		txPoolSize := cctx.Uint("tx-pool-size")

		r, err := initRepo(repoPath, chainAddress, txPoolSize)
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

		chainSvc, err := chain.NewChainSvc(ctx, chainAddress, "/websocket", cliutil.KeyringHome)
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

			coins, err := chainSvc.GetBalance(ctx, creator)
			if err != nil {
				fmt.Printf("%v", err)
				continue
			} else {
				if coins.AmountOf(chain.DENOM).LT(math.NewInt(int64(1100))) {
					continue
				} else {
					break
				}
			}

		}

		if tx, err := chainSvc.Create(ctx, creator); err != nil {
			// TODO: clear dir
			return err
		} else {
			fmt.Println(tx)
		}

		if txPoolSize > 0 {
			err = chain.CreateAddressPool(ctx, cliutil.KeyringHome, txPoolSize)
			if err != nil {
				return err
			}

			ap, err := chain.LoadAddressPool(ctx, cliutil.KeyringHome, txPoolSize)
			if err != nil {
				return err
			}

			for address := range ap.Addresses {
				amount := int64(1000 / txPoolSize)
				if tx, err := chainSvc.Send(ctx, creator, address, amount); err != nil {
					// TODO: clear dir
					return err
				} else {
					fmt.Printf("Sent %d SAO from creator %s to pool address %s, txhash=%s\r", amount, creator, address, tx)
				}
			}
		}

		return nil
	},
}

func initRepo(repoPath string, chainAddress string, TxPoolSize uint) (*repo.Repo, error) {
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
	if err = r.Init(chainAddress, TxPoolSize); err != nil {
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

		// update metadata datastore
		mds, err := repo.Datastore(ctx, "/metadata")
		if err != nil {
			return types.Wrap(types.ErrOpenDataStoreFailed, err)
		}
		if err := mds.Put(ctx, datastore.NewKey("node-address"), []byte(creator)); err != nil {
			return types.Wrap(types.ErrGetFailed, err)
		}

		tx, err := chain.Create(ctx, creator)
		if err != nil {
			return err
		} else {
			fmt.Println(tx)
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

var addStorageCmd = &cli.Command{
	Name:  "add-storage",
	Usage: "add storage",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "creator",
			Usage: "node's account on sao chain",
		},
		&cli.Uint64Flag{
			Name:     "size",
			Usage:    "storage size to add",
			Value:    1000,
			Required: false,
		},
	},
	Action: func(cctx *cli.Context) error {
		ctx := cctx.Context
		// TODO: validate input
		creator := cctx.String("creator")
		if creator == "" {
			return types.Wrapf(types.ErrInvalidParameters, "must provide --creator")
		}

		chainAddress, err := cliutil.GetChainAddress(cctx, cctx.String("repo"), cctx.App.Name)
		if err != nil {
			log.Warn(err)
		}

		chainSvc, err := chain.NewChainSvc(ctx, chainAddress, "/websocket", cliutil.KeyringHome)
		if err != nil {
			return err
		}

		size := cctx.Uint64("size")
		if size <= 0 {
			return types.Wrapf(types.ErrInvalidParameters, "invalid size")
		}

		tx, err := chainSvc.AddVstorage(ctx, creator, size)
		if err != nil {
			return err
		}
		fmt.Println(tx)

		return nil
	},
}

var removeStorageCmd = &cli.Command{
	Name:  "remove-storage",
	Usage: "remove storage",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "creator",
			Usage: "node's account on sao chain",
		},
		&cli.Uint64Flag{
			Name:     "size",
			Usage:    "storage size to remove",
			Value:    1000,
			Required: false,
		},
	},
	Action: func(cctx *cli.Context) error {
		ctx := cctx.Context
		// TODO: validate input
		creator := cctx.String("creator")
		if creator == "" {
			return types.Wrapf(types.ErrInvalidParameters, "must provide --creator")
		}

		chainAddress, err := cliutil.GetChainAddress(cctx, cctx.String("repo"), cctx.App.Name)
		if err != nil {
			log.Warn(err)
		}

		chainSvc, err := chain.NewChainSvc(ctx, chainAddress, "/websocket", cliutil.KeyringHome)
		if err != nil {
			return err
		}

		size := cctx.Uint64("size")
		if size <= 0 {
			return types.Wrapf(types.ErrInvalidParameters, "invalid size")
		}

		tx, err := chainSvc.RemoveVstorage(ctx, creator, size)
		if err != nil {
			return err
		}
		fmt.Println(tx)

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
		&cli.StringFlag{
			Name:  "details",
			Usage: "node's details informaton",
		},
		&cli.StringFlag{
			Name:  "identity",
			Usage: "keybase identity for the node",
		},
		&cli.StringFlag{
			Name:  "moniker",
			Usage: "node's moniker",
		},
		&cli.StringFlag{
			Name:  "security-contact",
			Usage: "node's security contact",
		},
		&cli.StringFlag{
			Name:  "website",
			Usage: "node's website",
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
				if strings.Contains(ma.String(), "127.0.0.1") {
					continue
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

		chainSvc, err := chain.NewChainSvc(ctx, chainAddress, "/websocket", cliutil.KeyringHome)
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

		var ap *chain.AddressPool
		if cfg.Chain.TxPoolSize > 0 {
			ap, err = chain.LoadAddressPool(ctx, cliutil.KeyringHome, cfg.Chain.TxPoolSize)
			if err != nil {
				return err
			}
		}

		addresses := make([]string, 0)
		for address := range ap.Addresses {
			addresses = append(addresses, address)
		}

		description := &nodetypes.Description{
			Details:         cctx.String("details"),
			Identity:        cctx.String("identity"),
			Moniker:         cctx.String("moniker"),
			SecurityContact: cctx.String("security-contact"),
			Website:         cctx.String("website"),
		}

		tx, err := chainSvc.Reset(ctx, creator, peerInfo, status, addresses, description)
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

		snode, err := node.NewNode(ctx, repo, cliutil.KeyringHome, cctx)
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

var declareFaultsRecoverCmd = &cli.Command{
	Name:  "declare-faults-recover",
	Usage: "declare that the storage faults have been recover and ready for check",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:     "creator",
			Usage:    "node's account on sao chain",
			Required: false,
		},
		&cli.Uint64SliceFlag{
			Name:     "shard-ids",
			Usage:    "shard id list to request for check",
			Required: true,
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

		shardIds := cctx.Uint64Slice("shard-ids")
		if len(shardIds) == 0 {
			return types.Wrapf(types.ErrInvalidParameters, "shard-ids is required")
		}

		chainAddress, err := cliutil.GetChainAddress(cctx, cctx.String("repo"), cctx.App.Name)
		if err != nil {
			log.Warn(err)
		}

		chain, err := chain.NewChainSvc(ctx, chainAddress, "/websocket", cliutil.KeyringHome)
		if err != nil {
			return err
		}

		faults := make([]*saotypes.Fault, 0)
		for _, shardId := range shardIds {
			shard, err := chain.GetShard(ctx, shardId)
			if err != nil {
				return err
			}
			order, err := chain.GetOrder(ctx, shard.OrderId)
			if err != nil {
				return err
			}

			faults = append(faults, &saotypes.Fault{
				DataId:   order.DataId,
				OrderId:  order.Id,
				ShardId:  shardId,
				CommitId: strings.Split(order.Commit, "\032")[0],
				Provider: creator,
				Reporter: creator,
			})
		}

		if tx, err := chain.RecoverFaults(ctx, creator, creator, faults); err != nil {
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
