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

var modelCmd = &cli.Command{
	Name:  "model",
	Usage: "data model management",
	Subcommands: []*cli.Command{
		createCmd,
		patchGenCmd,
		updateCmd,
		loadCmd,
		deleteCmd,
		commitsCmd,
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
				`{"alias": "%s", "dataId": "%s", "ExtendInfo": "%s", "groupId": "%s", "commit": "%s", "update": false}`,
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
			Name:     "version",
			Usage:    "data model's version",
			Required: false,
		},
		&cli.StringFlag{
			Name:     "commit-id",
			Usage:    "data model's commitId",
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

		version := cctx.String("version")
		commitId := cctx.String("commit-id")
		if cctx.IsSet("version") && cctx.IsSet("commit-id") {
			log.Warn("--version is to be ignored once --commit-id is specified")
			version = ""
		}

		client := saoclient.NewSaoClient(gatewayApi)
		groupId := cctx.String("platform")
		if groupId == "" {
			groupId = client.Cfg.GroupId
		}

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

var commitsCmd = &cli.Command{
	Name:  "commits",
	Usage: "list data model historical commits",
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

		resp, err := client.ShowCommits(ctx, owner, key, groupId)
		if err != nil {
			return err
		}
		log.Infof("Model[%s] - %s", resp.DataId, resp.Alias)
		log.Info("---------------------------------------------------------------")
		log.Infof("Version\tCommit                              \tHeight")
		for i, commit := range resp.Commits {
			commitInfo := strings.Split(commit, "\032")
			if len(commitInfo) != 2 || len(commitInfo[1]) == 0 {
				return xerrors.Errorf("invalid commit information: %s", commit)
			}

			log.Infof("v%d\t\t|%s|%s", i, commitInfo[0], commitInfo[1])
		}
		log.Info("---------------------------------------------------------------")

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

			metadata := fmt.Sprintf(`{"dataId": "%s", "commit": "%s", "update": true}`, orderMeta.DataId, orderMeta.CommitId)
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

		origin := cctx.String("origin")
		target := cctx.String("target")
		patch, err := utils.GeneratePatch(origin, target)
		if err != nil {
			return err
		}

		content, err := utils.ApplyPatch([]byte(origin), []byte(patch))
		if err != nil {
			return err
		}

		targetCid, err := utils.CaculateCid(content)
		if err != nil {
			return err
		}

		log.Info("Patch: ", patch)
		log.Info("Target cid: ", targetCid)

		return nil
	},
}
