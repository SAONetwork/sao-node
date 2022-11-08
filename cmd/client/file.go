package main

import (
	"fmt"
	"os"
	"path/filepath"
	apiclient "sao-storage-node/api/client"
	saoclient "sao-storage-node/client"
	"sao-storage-node/node/chain"
	"sao-storage-node/types"
	"sao-storage-node/utils"
	"strings"

	"github.com/ipfs/go-cid"
	"github.com/urfave/cli/v2"
	"golang.org/x/xerrors"
)

var fileCmd = &cli.Command{
	Name:  "file",
	Usage: "file management",
	Subcommands: []*cli.Command{
		createFileCmd,
		peerInfoCmd,
		uploadCmd,
		downloadCmd,
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
				`{"alias": "%s", "dataId": "%s", "ExtendInfo": "%s", "groupId": "%s", "commit": "%s", "update": false}`,
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

var downloadCmd = &cli.Command{
	Name:  "download",
	Usage: "download file(s) from storage network",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:     "owner",
			Usage:    "file owner",
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
			Name:     "version",
			Usage:    "file version",
			Required: false,
		},
		&cli.StringFlag{
			Name:     "commit-id",
			Usage:    "file commitId",
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

		version := cctx.String("version")
		commitId := cctx.String("commit-id")
		if cctx.IsSet("version") && cctx.IsSet("commit-id") {
			log.Warn("--version is to be ignored once --commit-id is specified")
			version = ""
		}

		for _, key := range keys {
			orderMeta := types.OrderMeta{
				Owner:    owner,
				GroupId:  groupId,
				DataId:   key,
				Alias:    key,
				CommitId: commitId,
				Version:  version,
				IsUpdate: false,
			}

			resp, err := client.Load(ctx, orderMeta)
			if err != nil {
				return err
			}

			log.Info("File DataId: ", resp.DataId)
			log.Info("File Name: ", resp.Alias)
			log.Info("File CommitId: ", resp.CommitId)
			log.Info("File Version: ", resp.Version)
			log.Info("File Cid: ", resp.Cid)
			log.Debugf("File Content: ", resp.Content)

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
