package main

// TODO:
// * how to generate cid from scratch
// * guic transfer data

import (
	"fmt"
	"os"
	"path/filepath"
	apiclient "sao-storage-node/api/client"
	"sao-storage-node/build"
	saoclient "sao-storage-node/client"
	cliutil "sao-storage-node/cmd"
	"sao-storage-node/node/chain"
	"sao-storage-node/types"
	"sao-storage-node/utils"
	"strings"

	"github.com/ipfs/go-cid"
	logging "github.com/ipfs/go-log/v2"
	"github.com/urfave/cli/v2"
	"golang.org/x/xerrors"
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
			createCmd,
			createFileCmd,
			deleteCmd,
			uploadCmd,
			loadCmd,
			updateCmd,
			downloadCmd,
			patchGenCmd,
			peerInfoCmd,
		},
	}
	app.Setup()

	if err := app.Run(os.Args); err != nil {
		os.Stderr.WriteString("Error: " + err.Error() + "\n")
		os.Exit(1)
	}
}

var testCmd = &cli.Command{
	Name: "test",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:     "gateway",
			Value:    "http://127.0.0.1:8888/rpc/v0",
			EnvVars:  []string{"SAO_GATEWAY_API"},
			Required: false,
		},
	},
	Action: func(cctx *cli.Context) error {
		ctx := cctx.Context
		gateway := cctx.String("gateway")
		gatewayApi, closer, err := apiclient.NewGatewayApi(ctx, gateway, nil)
		if err != nil {
			return err
		}
		defer closer()

		client := saoclient.NewSaoClient(gatewayApi)
		resp, err := client.Test(ctx)
		if err != nil {
			return err
		}
		log.Info(resp)
		return nil
	},
}

var createCmd = &cli.Command{
	Name: "create",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:     "owner",
			Usage:    "data model owner",
			Required: true,
		},
		&cli.StringFlag{
			Name:     "platform",
			Usage:    "platform to manage the data model",
			Required: false,
		},
		&cli.StringFlag{
			Name:     "content",
			Required: false,
		},
		&cli.IntFlag{
			Name:     "duration",
			Usage:    "how long do you want to store the data.",
			Value:    DEFAULT_DURATION,
			Required: false,
		},
		&cli.BoolFlag{
			Name:     "clientPublish",
			Value:    false,
			Required: false,
		},
		&cli.StringFlag{
			Name:     "chainAddress",
			Value:    "http://localhost:26657",
			Required: false,
		},
		&cli.StringFlag{
			Name:     "name",
			Value:    "",
			Required: false,
		},
		&cli.StringFlag{
			Name:     "cid",
			Value:    "",
			Required: false,
		},
		&cli.IntFlag{
			Name:     "replica",
			Usage:    "how many copies to store.",
			Value:    DEFAULT_REPLICA,
			Required: false,
		},
		&cli.StringFlag{
			Name:     "gateway",
			Value:    "http://127.0.0.1:8888/rpc/v0",
			EnvVars:  []string{"SAO_GATEWAY_API"},
			Required: false,
		},
		&cli.StringFlag{
			Name:     "extend-info",
			Usage:    "extend information for the model",
			Value:    "",
			Required: false,
		},
	},
	Action: func(cctx *cli.Context) error {
		ctx := cctx.Context

		// ---- check parameters ----
		if !cctx.IsSet("content") || cctx.String("content") == "" {
			return xerrors.Errorf("must provide non-empty --content.")
		}
		content := []byte(cctx.String("content"))

		if !cctx.IsSet("owner") {
			return xerrors.Errorf("must provide --owner")
		}
		owner := cctx.String("owner")
		clientPublish := cctx.Bool("clientPublish")

		// TODO: check valid range
		duration := cctx.Int("duration")
		replicas := cctx.Int("replica")
		chainAddress := cctx.String("chainAddress")

		gateway := cctx.String("gateway")
		gatewayApi, closer, err := apiclient.NewGatewayApi(ctx, gateway, nil)
		if err != nil {
			return err
		}
		defer closer()

		extendInfo := cctx.String("extend-info")
		if len(extendInfo) > 1024 {
			return xerrors.Errorf("extend-info should no longer than 1024 characters")
		}

		client := saoclient.NewSaoClient(gatewayApi)
		groupId := cctx.String("platform")
		if groupId == "" {
			groupId = client.Cfg.GroupId
		}

		orderMeta := types.OrderMeta{
			Owner:      owner,
			GroupId:    groupId,
			Alias:      cctx.String("name"),
			Duration:   int32(duration),
			Replica:    int32(replicas),
			ExtendInfo: extendInfo,
			IsUpdate:   false,
		}

		contentCid, err := utils.CaculateCid(content)
		if err != nil {
			return err
		}
		orderMeta.Cid = contentCid

		if clientPublish {
			gatewayAddress, err := gatewayApi.NodeAddress(ctx)
			if err != nil {
				return err
			}

			chain, err := chain.NewChainSvc(ctx, "cosmos", chainAddress, "/websocket")
			if err != nil {
				return xerrors.Errorf("new cosmos chain: %w", err)
			}

			orderMeta.DataId = utils.GenerateDataId()
			orderMeta.CommitId = orderMeta.DataId
			metadata := fmt.Sprintf(
				`{"alias": "%s", "dataId": "%s", "ExtendInfo": "%s", "groupId": "%s", "commitId": "%s", "update": false}`,
				orderMeta.Alias,
				orderMeta.DataId,
				orderMeta.ExtendInfo,
				orderMeta.GroupId,
				orderMeta.CommitId,
			)
			log.Info("metadata: ", metadata)

			orderId, tx, err := chain.StoreOrder(ctx, owner, owner, gatewayAddress, contentCid, int32(duration), int32(replicas), metadata)
			if err != nil {
				return err
			}
			log.Infof("order id=%d, tx=%s", orderId, tx)
			orderMeta.TxId = tx
			orderMeta.OrderId = orderId
			orderMeta.TxSent = true
		}

		resp, err := client.Create(ctx, orderMeta, content)
		if err != nil {
			return err
		}
		log.Infof("alias: %s, data id: %s", resp.Alias, resp.DataId)
		return nil
	},
}

var createFileCmd = &cli.Command{
	Name: "create-file",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:     "owner",
			Usage:    "file owner",
			Required: true,
		},
		&cli.StringFlag{
			Name:     "platform",
			Usage:    "platform to manage the file",
			Required: false,
		},
		&cli.StringFlag{
			Name:     "file-name",
			Required: true,
		},
		&cli.IntFlag{
			Name:     "duration",
			Usage:    "how long do you want to store the data.",
			Value:    DEFAULT_DURATION,
			Required: false,
		},
		&cli.BoolFlag{
			Name:     "clientPublish",
			Value:    false,
			Required: false,
		},
		&cli.StringFlag{
			Name:     "chainAddress",
			Value:    "http://localhost:26657",
			Required: false,
		},
		&cli.StringFlag{
			Name:     "cid",
			Value:    "",
			Required: true,
		},
		&cli.IntFlag{
			Name:     "replica",
			Usage:    "how many copies to store.",
			Value:    DEFAULT_REPLICA,
			Required: false,
		},
		&cli.StringFlag{
			Name:     "gateway",
			Value:    "http://127.0.0.1:8888/rpc/v0",
			EnvVars:  []string{"SAO_GATEWAY_API"},
			Required: false,
		},
		&cli.StringFlag{
			Name:     "extend-info",
			Usage:    "extend information for the model",
			Value:    "",
			Required: false,
		},
	},
	Action: func(cctx *cli.Context) error {
		ctx := cctx.Context

		// ---- check parameters ----
		// if !cctx.IsSet("content") || cctx.String("content") == "" {
		// 	return xerrors.Errorf("must provide non-empty --content.")
		// }
		if !cctx.IsSet("owner") {
			return xerrors.Errorf("must provide --owner")
		}
		owner := cctx.String("owner")

		if !cctx.IsSet("file-name") {
			return xerrors.Errorf("must provide --file-name")
		}
		fileName := cctx.String("file-name")

		clientPublish := cctx.Bool("clientPublish")

		// TODO: check valid range
		duration := cctx.Int("duration")
		replicas := cctx.Int("replica")
		chainAddress := cctx.String("chainAddress")

		gateway := cctx.String("gateway")
		gatewayApi, closer, err := apiclient.NewGatewayApi(ctx, gateway, nil)
		if err != nil {
			return err
		}
		defer closer()

		extendInfo := cctx.String("extend-info")
		if len(extendInfo) > 1024 {
			return xerrors.Errorf("extend-info should no longer than 1024 characters")
		}

		client := saoclient.NewSaoClient(gatewayApi)
		groupId := cctx.String("platform")
		if groupId == "" {
			groupId = client.Cfg.GroupId
		}

		orderMeta := types.OrderMeta{
			Owner:      owner,
			GroupId:    groupId,
			Alias:      fileName,
			Duration:   int32(duration),
			Replica:    int32(replicas),
			ExtendInfo: extendInfo,
			IsUpdate:   false,
		}
		// TODO:
		cid, err := cid.Decode(cctx.String("cid"))
		if err == nil {
			orderMeta.Cid = cid
		}

		if clientPublish {
			gatewayAddress, err := gatewayApi.NodeAddress(ctx)
			if err != nil {
				return err
			}

			chain, err := chain.NewChainSvc(ctx, "cosmos", chainAddress, "/websocket")
			if err != nil {
				return xerrors.Errorf("new cosmos chain: %w", err)
			}

			orderMeta.DataId = utils.GenerateDataId()
			orderMeta.CommitId = orderMeta.DataId
			metadata := fmt.Sprintf(
				`{"alias": "%s", "dataId": "%s", "ExtendInfo": "%s", "groupId": "%s", "commitId": "%s", "update": false}`,
				orderMeta.Alias,
				orderMeta.DataId,
				orderMeta.ExtendInfo,
				orderMeta.GroupId,
				orderMeta.CommitId,
			)
			log.Info("metadata: ", metadata)

			orderId, tx, err := chain.StoreOrder(ctx, owner, owner, gatewayAddress, cid, int32(duration), int32(replicas), metadata)
			if err != nil {
				return err
			}
			log.Infof("order id=%d, tx=%s", orderId, tx)
			orderMeta.TxId = tx
			orderMeta.OrderId = orderId
			orderMeta.TxSent = true
		}

		resp, err := client.CreateFile(ctx, orderMeta)
		if err != nil {
			return err
		}
		log.Infof("file name: %s, data id: %s, cid: %v", resp.Alias, resp.DataId, resp.Cid)
		return nil
	},
}

var uploadCmd = &cli.Command{
	Name:  "upload",
	Usage: "upload file(s) to storage network",
	Flags: []cli.Flag{
		&cli.PathFlag{
			Name:     "filepath",
			Usage:    "file's path to upload",
			Required: true,
		},
		&cli.StringFlag{
			Name:     "multiaddr",
			Usage:    "remote multiaddr",
			Required: true,
		},
	},
	Action: func(cctx *cli.Context) error {
		ctx := cctx.Context

		fpath := cctx.String("filepath")
		multiaddr := cctx.String("multiaddr")
		if !strings.Contains(multiaddr, "/p2p/") {
			return fmt.Errorf("invalid multiaddr: %s", multiaddr)
		}
		peerId := strings.Split(multiaddr, "/p2p/")[1]

		var files []string
		err := filepath.Walk(fpath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if !info.IsDir() {
				files = append(files, path)
			} else {
				log.Warn("skip directory ", path)
			}

			return nil
		})

		if err != nil {
			log.Fatal(err)
		}

		for _, file := range files {
			log.Info("uploading file ", file)

			c := saoclient.DoTransport(ctx, multiaddr, peerId, file)
			if c != cid.Undef {
				log.Info("file [", file, "] successfully uploaded, CID is ", c.String())
			} else {
				log.Warn("failed to uploaded the file [", file, "], please try again")
			}
		}

		return nil
	},
}

var loadCmd = &cli.Command{
	Name:  "load",
	Usage: "load data model",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:     "owner",
			Usage:    "data model's owner",
			Required: true,
		},
		&cli.StringFlag{
			Name:     "key",
			Usage:    "data model's alias, dataId or tag",
			Required: true,
		},
		&cli.StringFlag{
			Name:     "platform",
			Usage:    "platform to manage the data model",
			Required: false,
		},
		&cli.StringFlag{
			Name:     "gateway",
			Value:    "http://127.0.0.1:8888/rpc/v0",
			EnvVars:  []string{"SAO_GATEWAY_API"},
			Required: false,
		},
		&cli.BoolFlag{
			Name:     "dump",
			Value:    false,
			Usage:    "dump data model content to current path",
			Required: false,
		},
	},
	Action: func(cctx *cli.Context) error {
		ctx := cctx.Context

		gateway := cctx.String("gateway")
		gatewayApi, closer, err := apiclient.NewGatewayApi(ctx, gateway, nil)
		if err != nil {
			return err
		}
		defer closer()

		if !cctx.IsSet("owner") {
			return xerrors.Errorf("must provide --owner")
		}
		owner := cctx.String("owner")

		if !cctx.IsSet("key") {
			return xerrors.Errorf("must provide --key")
		}
		key := cctx.String("key")

		client := saoclient.NewSaoClient(gatewayApi)
		groupId := cctx.String("platform")
		if groupId == "" {
			groupId = client.Cfg.GroupId
		}

		resp, err := client.Load(ctx, owner, key, groupId)
		if err != nil {
			return err
		}
		log.Infof("alias id: %s, data id: %s, content: %s", resp.Alias, resp.DataId, resp.Content)

		dumpFlag := cctx.Bool("dump")
		if dumpFlag {
			path := filepath.Join("./", resp.DataId+".json")
			file, err := os.Create(path)
			if err != nil {
				return err
			}

			_, err = file.Write([]byte(resp.Content))
			if err != nil {
				return err
			}
			log.Infof("data model dumped to %s", path)
		}

		return nil
	},
}

var deleteCmd = &cli.Command{
	Name:  "delete",
	Usage: "delete data model",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:     "owner",
			Usage:    "data model's owner",
			Required: true,
		},
		&cli.StringFlag{
			Name:     "key",
			Usage:    "data model's alias, dataId or tag",
			Required: true,
		},
		&cli.StringFlag{
			Name:     "platform",
			Usage:    "platform to manage the data model",
			Required: false,
		},
		&cli.StringFlag{
			Name:     "gateway",
			Value:    "http://127.0.0.1:8888/rpc/v0",
			EnvVars:  []string{"SAO_GATEWAY_API"},
			Required: false,
		},
	},
	Action: func(cctx *cli.Context) error {
		ctx := cctx.Context

		gateway := cctx.String("gateway")
		gatewayApi, closer, err := apiclient.NewGatewayApi(ctx, gateway, nil)
		if err != nil {
			return err
		}
		defer closer()

		if !cctx.IsSet("owner") {
			return xerrors.Errorf("must provide --owner")
		}
		owner := cctx.String("owner")

		if !cctx.IsSet("key") {
			return xerrors.Errorf("must provide --key")
		}
		key := cctx.String("key")

		client := saoclient.NewSaoClient(gatewayApi)
		groupId := cctx.String("platform")
		if groupId == "" {
			groupId = client.Cfg.GroupId
		}

		resp, err := client.Delete(ctx, owner, key, groupId)
		if err != nil {
			return err
		}
		log.Infof("data model %s deleted", resp.Alias)

		return nil
	},
}

var downloadCmd = &cli.Command{
	Name:  "download",
	Usage: "download file(s) from storage network",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:     "owner",
			Usage:    "data model's owner",
			Required: true,
		},
		&cli.StringSliceFlag{
			Name:     "keys",
			Usage:    "storage network dataId(s) of the file(s)",
			Required: true,
		},
		&cli.StringFlag{
			Name:     "platform",
			Usage:    "platform to manage the data model",
			Required: false,
		},
		&cli.StringFlag{
			Name:     "gateway",
			Value:    "http://127.0.0.1:8888/rpc/v0",
			EnvVars:  []string{"SAO_GATEWAY_API"},
			Required: false,
		},
	},
	Action: func(cctx *cli.Context) error {
		ctx := cctx.Context

		gateway := cctx.String("gateway")
		gatewayApi, closer, err := apiclient.NewGatewayApi(ctx, gateway, nil)
		if err != nil {
			return err
		}
		defer closer()

		if !cctx.IsSet("owner") {
			return xerrors.Errorf("must provide --owner")
		}
		owner := cctx.String("owner")

		if !cctx.IsSet("keys") {
			return xerrors.Errorf("must provide --keys")
		}
		keys := cctx.StringSlice("keys")

		client := saoclient.NewSaoClient(gatewayApi)

		groupId := cctx.String("platform")
		if groupId == "" {
			groupId = client.Cfg.GroupId
		}

		for _, key := range keys {
			resp, err := client.Load(ctx, owner, key, groupId)
			if err != nil {
				return err
			}
			log.Infof("file name: %d, data id: %s", resp.Alias, resp.DataId)

			path := filepath.Join("./", resp.Alias)
			file, err := os.Create(path)
			if err != nil {
				return err
			}

			_, err = file.Write([]byte(resp.Content))
			if err != nil {
				return err
			}
			log.Infof("file downloaded to %s", path)
		}

		return nil
	},
}

var updateCmd = &cli.Command{
	Name: "update",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:     "owner",
			Usage:    "data model owner",
			Required: true,
		},
		&cli.StringFlag{
			Name:     "platform",
			Usage:    "platform to manage the data model",
			Required: false,
		},
		&cli.StringFlag{
			Name:     "patch",
			Usage:    "patch to apply for the data model",
			Required: true,
		},
		&cli.IntFlag{
			Name:     "duration",
			Usage:    "how long do you want to store the data.",
			Value:    DEFAULT_DURATION,
			Required: false,
		},
		&cli.BoolFlag{
			Name:     "clientPublish",
			Value:    false,
			Required: false,
		},
		&cli.StringFlag{
			Name:     "chainAddress",
			Value:    "http://localhost:26657",
			Required: false,
		},
		&cli.StringFlag{
			Name:     "data-id",
			Required: false,
		},
		&cli.StringFlag{
			Name:     "name",
			Required: false,
		},
		&cli.StringFlag{
			Name:     "cid",
			Required: true,
		},
		&cli.IntFlag{
			Name:     "replica",
			Usage:    "how many copies to store.",
			Value:    DEFAULT_REPLICA,
			Required: false,
		},
		&cli.StringFlag{
			Name:     "gateway",
			Value:    "http://127.0.0.1:8888/rpc/v0",
			EnvVars:  []string{"SAO_GATEWAY_API"},
			Required: false,
		},
		&cli.StringFlag{
			Name:     "extend-info",
			Usage:    "extend information for the model",
			Required: false,
		},
	},
	Action: func(cctx *cli.Context) error {
		ctx := cctx.Context

		// ---- check parameters ----
		if !cctx.IsSet("data-id") && !cctx.IsSet("name") {
			return xerrors.Errorf("please provide either --data-id or --name")
		}

		patch := []byte(cctx.String("patch"))
		contentCid := cctx.String("cid")
		newCid, err := cid.Decode(contentCid)
		if err != nil {
			return err
		}

		owner := cctx.String("owner")

		extendInfo := cctx.String("extend-info")
		if len(extendInfo) > 1024 {
			return xerrors.Errorf("extend-info should no longer than 1024 characters")
		}

		clientPublish := cctx.Bool("clientPublish")

		// TODO: check valid range
		duration := cctx.Int("duration")
		replicas := cctx.Int("replica")
		chainAddress := cctx.String("chainAddress")

		gateway := cctx.String("gateway")
		gatewayApi, closer, err := apiclient.NewGatewayApi(ctx, gateway, nil)
		if err != nil {
			return err
		}
		defer closer()

		client := saoclient.NewSaoClient(gatewayApi)
		groupId := cctx.String("platform")
		if groupId == "" {
			groupId = client.Cfg.GroupId
		}

		orderMeta := types.OrderMeta{
			Owner:      owner,
			GroupId:    groupId,
			DataId:     cctx.String("data-id"),
			Alias:      cctx.String("name"),
			Duration:   int32(duration),
			Replica:    int32(replicas),
			ExtendInfo: extendInfo,
			Cid:        newCid,
			IsUpdate:   true,
		}

		if clientPublish {
			gatewayAddress, err := gatewayApi.NodeAddress(ctx)
			if err != nil {
				return err
			}

			chain, err := chain.NewChainSvc(ctx, "cosmos", chainAddress, "/websocket")
			if err != nil {
				return xerrors.Errorf("new cosmos chain: %w", err)
			}

			key := orderMeta.DataId
			if key != "" {
				key = orderMeta.Alias
			}
			meta, err := chain.QueryMeta(ctx, key, 0)
			if err != nil {
				return err
			}
			log.Debugf("meta: DataId=%s, Alias=%s", meta.Metadata.DataId, meta.Metadata.Alias)
			orderMeta.Alias = meta.Metadata.Alias
			orderMeta.DataId = meta.Metadata.DataId
			orderMeta.CommitId = utils.GenerateCommitId()

			metadata := fmt.Sprintf(`{"dataId": "%s", "commitId": "%s", "update": true}`, orderMeta.DataId, orderMeta.CommitId)
			log.Info("metadata: ", metadata)

			orderId, tx, err := chain.StoreOrder(ctx, owner, owner, gatewayAddress, orderMeta.Cid, int32(duration), int32(replicas), metadata)
			if err != nil {
				return err
			}
			log.Infof("order id=%d, tx=%s", orderId, tx)
			orderMeta.TxId = tx
			orderMeta.OrderId = orderId
			orderMeta.TxSent = true
		}

		resp, err := client.Update(ctx, orderMeta, patch)
		if err != nil {
			return err
		}
		log.Infof("alias: %s, data id: %s", resp.Alias, resp.DataId)
		return nil
	},
}

var patchGenCmd = &cli.Command{
	Name:  "patch-gen",
	Usage: "generate data model patch",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:     "origin",
			Usage:    "the original data model\r\n",
			Required: true,
		},
		&cli.StringFlag{
			Name:     "target",
			Usage:    "the target data model\r\n",
			Required: true,
		},
	},
	Action: func(cctx *cli.Context) error {
		if !cctx.IsSet("origin") || !cctx.IsSet("target") {
			return xerrors.Errorf("please provide both --origin and --target")
		}

		patch, err := saoclient.GeneratePatch(cctx.String("origin"), cctx.String("target"))
		if err != nil {
			return err
		}

		targetCid, err := utils.CaculateCid([]byte(cctx.String("target")))
		if err != nil {
			return err
		}

		log.Info("Patch: ", patch)
		log.Info("Target cid: ", targetCid)

		return nil
	},
}

var peerInfoCmd = &cli.Command{
	Name:  "peer-info",
	Usage: "get peer info of the gateway",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:     "gateway",
			Value:    "http://127.0.0.1:8888/rpc/v0",
			EnvVars:  []string{"SAO_GATEWAY_API"},
			Required: false,
		},
	},
	Action: func(cctx *cli.Context) error {
		ctx := cctx.Context

		gateway := cctx.String("gateway")
		gatewayApi, closer, err := apiclient.NewGatewayApi(ctx, gateway, nil)
		if err != nil {
			return err
		}
		defer closer()

		client := saoclient.NewSaoClient(gatewayApi)
		resp, err := client.GetPeerInfo(ctx)
		if err != nil {
			return err
		}
		log.Info("peer info: ", resp.PeerInfo)

		return nil
	},
}
