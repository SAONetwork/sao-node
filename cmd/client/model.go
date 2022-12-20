package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sao-node/chain"
	cliutil "sao-node/cmd"
	"sao-node/types"
	"sao-node/utils"
	"strconv"
	"strings"
	"time"

	did "github.com/SaoNetwork/sao-did"
	saotypes "github.com/SaoNetwork/sao/x/sao/types"
	"github.com/fatih/color"
	"github.com/ipfs/go-cid"
	"github.com/urfave/cli/v2"

	"golang.org/x/xerrors"
)

var modelCmd = &cli.Command{
	Name:      "model",
	Usage:     "data model management",
	UsageText: "model related commands including create, update, update permission, etc.",
	Subcommands: []*cli.Command{
		createCmd,
		patchGenCmd,
		updateCmd,
		updatePermissionCmd,
		loadCmd,
		deleteCmd,
		commitsCmd,
		renewCmd,
		statusCmd,
	},
}

var createCmd = &cli.Command{
	Name:  "create",
	Usage: "create a new data model",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:     "content",
			Required: false,
			Usage:    "data model content to create. you must either specify --content or --cid",
		},
		&cli.StringFlag{
			Name:     "cid",
			Usage:    "data content cid, make sure gateway has this cid file before using this flag. you must either specify --content or --cid. ",
			Value:    "",
			Required: false,
		},
		&cli.IntFlag{
			Name:     "duration",
			Usage:    "how many days do you want to store the data",
			Value:    DEFAULT_DURATION,
			Required: false,
		},
		&cli.IntFlag{
			Name:     "delay",
			Usage:    "how many epochs to wait for the content to be completed storing",
			Value:    1 * 60,
			Required: false,
		},
		&cli.BoolFlag{
			Name:     "client-publish",
			Usage:    "true if client sends MsgStore message on chain, or leave it to gateway to send",
			Value:    false,
			Required: false,
		},
		&cli.StringFlag{
			Name:     "name",
			Usage:    "alias name for this data model, this alias name can be used to update, load, etc.",
			Value:    "",
			Required: false,
		},
		&cli.StringSliceFlag{
			Name:     "tags",
			Required: false,
		},
		&cli.StringFlag{
			Name:     "rule",
			Value:    "",
			Required: false,
		},
		&cli.IntFlag{
			Name:     "replica",
			Usage:    "how many copies to store",
			Value:    DEFAULT_REPLICA,
			Required: false,
		},
		&cli.StringFlag{
			Name:     "extend-info",
			Usage:    "extend information for the model",
			Value:    "",
			Required: false,
		},
		&cli.BoolFlag{
			Name:     "public",
			Value:    false,
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

		clientPublish := cctx.Bool("client-publish")

		// TODO: check valid range
		duration := cctx.Int("duration")
		replicas := cctx.Int("replica")
		delay := cctx.Int("delay")
		isPublic := cctx.Bool("public")

		extendInfo := cctx.String("extend-info")
		if len(extendInfo) > 1024 {
			return xerrors.Errorf("extend-info should no longer than 1024 characters")
		}

		client, closer, err := getSaoClient(cctx)
		if err != nil {
			return err
		}
		defer closer()

		if client == nil {
			return xerrors.Errorf("failed to create client")
		}

		groupId := cctx.String("platform")
		if groupId == "" {
			groupId = client.Cfg.GroupId
		}

		contentCid, err := utils.CalculateCid(content)
		if err != nil {
			return err
		}

		didManager, signer, err := cliutil.GetDidManager(cctx, client.Cfg)
		if err != nil {
			return err
		}

		gatewayAddress, err := client.GetNodeAddress(ctx)
		if err != nil {
			return err
		}

		dataId := utils.GenerateDataId()
		proposal := saotypes.Proposal{
			DataId:   dataId,
			Owner:    didManager.Id,
			Provider: gatewayAddress,
			GroupId:  groupId,
			Duration: uint64(time.Duration(60*60*24*duration) * time.Second / chain.Blocktime),
			Replica:  int32(replicas),
			Timeout:  int32(delay),
			Alias:    cctx.String("name"),
			Tags:     cctx.StringSlice("tags"),
			Cid:      contentCid.String(),
			CommitId: dataId,
			Rule:     cctx.String("rule"),
			// OrderId:    0,
			Size_:      uint64(len(content)),
			Operation:  1,
			ExtendInfo: extendInfo,
		}
		if proposal.Alias == "" {
			proposal.Alias = proposal.Cid
		}

		queryProposal := saotypes.QueryProposal{
			Owner:   didManager.Id,
			Keyword: dataId,
		}

		if isPublic {
			queryProposal.Owner = "all"
			proposal.Owner = "all"
		}

		clientProposal, err := buildClientProposal(ctx, didManager, proposal, client)
		if err != nil {
			return err
		}

		var orderId uint64 = 0
		if clientPublish {
			resp, _, _, err := client.StoreOrder(ctx, signer, clientProposal)
			if err != nil {
				return err
			}
			orderId = resp.OrderId
		}

		request, err := buildQueryRequest(ctx, didManager, queryProposal, client, gatewayAddress)
		if err != nil {
			return err
		}

		resp, err := client.ModelCreate(ctx, request, clientProposal, orderId, content)
		if err != nil {
			return err
		}
		fmt.Printf("alias: %s, data id: %s\r\n", resp.Alias, resp.DataId)
		return nil
	},
}

var loadCmd = &cli.Command{
	Name:      "load",
	Usage:     "load data model",
	UsageText: "only owner and dids with r/rw permission can load data model.",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:     "keyword",
			Usage:    "data model's alias, dataId or tag",
			Required: true,
		},
		&cli.StringFlag{
			Name:     "version",
			Usage:    "data model's version. you can find out version in commits cmd",
			Required: false,
		},
		&cli.StringFlag{
			Name:     "commit-id",
			Usage:    "data model's commitId",
			Required: false,
		},
		&cli.BoolFlag{
			Name:     "dump",
			Value:    false,
			Usage:    "dump data model content to ./<dataid>.json",
			Required: false,
		},
	},
	Action: func(cctx *cli.Context) error {
		ctx := cctx.Context

		if !cctx.IsSet("keyword") {
			return xerrors.Errorf("must provide --keyword")
		}
		keyword := cctx.String("keyword")

		version := cctx.String("version")
		commitId := cctx.String("commit-id")
		if cctx.IsSet("version") && cctx.IsSet("commit-id") {
			fmt.Println("--version is to be ignored once --commit-id is specified")
			version = ""
		}

		client, closer, err := getSaoClient(cctx)
		if err != nil {
			return err
		}
		defer closer()

		groupId := cctx.String("platform")
		if groupId == "" {
			groupId = client.Cfg.GroupId
		}

		didManager, _, err := cliutil.GetDidManager(cctx, client.Cfg)
		if err != nil {
			return err
		}

		proposal := saotypes.QueryProposal{
			Owner:    didManager.Id,
			Keyword:  keyword,
			GroupId:  groupId,
			CommitId: commitId,
			Version:  version,
		}

		if !utils.IsDataId(keyword) {
			proposal.KeywordType = 2
		}

		gatewayAddress, err := client.GetNodeAddress(ctx)
		if err != nil {
			return err
		}

		request, err := buildQueryRequest(ctx, didManager, proposal, client, gatewayAddress)
		if err != nil {
			return err
		}

		resp, err := client.ModelLoad(ctx, request)
		if err != nil {
			return err
		}

		console := color.New(color.FgMagenta, color.Bold)

		fmt.Print("  DataId    : ")
		console.Println(resp.DataId)

		fmt.Print("  Alias     : ")
		console.Println(resp.Alias)

		fmt.Print("  CommitId  : ")
		console.Println(resp.CommitId)

		fmt.Print("  Version   : ")
		console.Println(resp.Version)

		fmt.Print("  Cid       : ")
		console.Println(resp.Cid)

		match, err := regexp.Match("^"+types.Type_Prefix_File, []byte(resp.Alias))
		if err != nil {
			return err
		}

		if len(resp.Content) == 0 || match {
			fmt.Print("  SAO Link  : ")
			console.Println("sao://" + resp.DataId)

			httpUrl, err := client.GetHttpUrl(ctx, resp.DataId)
			if err != nil {
				return err
			}
			fmt.Print("  HTTP Link : ")
			console.Println(httpUrl.Url)

			ipfsUrl, err := client.GetIpfsUrl(ctx, resp.Cid)
			if err != nil {
				return err
			}
			fmt.Print("  IPFS Link : ")
			console.Println(ipfsUrl.Url)
		} else {
			fmt.Print("  Content   : ")
			console.Println(resp.Content)
		}

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
			fmt.Printf("data model dumped to %s.\r\n", path)
		}

		return nil
	},
}

var renewCmd = &cli.Command{
	Name:  "renew",
	Usage: "renew data model",
	Flags: []cli.Flag{
		&cli.StringSliceFlag{
			Name:     "data-ids",
			Usage:    "data model's dataId list",
			Required: true,
		},
		&cli.IntFlag{
			Name:     "duration",
			Usage:    "how many days do you want to renew the data.",
			Value:    DEFAULT_DURATION,
			Required: false,
		},
		&cli.IntFlag{
			Name:     "delay",
			Usage:    "how long to wait for the file ready",
			Value:    1 * 60,
			Required: false,
		},
		&cli.BoolFlag{
			Name:     "client-publish",
			Usage:    "true if client sends MsgStore message on chain, or leave it to gateway to send",
			Value:    false,
			Required: false,
		},
	},
	Action: func(cctx *cli.Context) error {
		ctx := cctx.Context

		if !cctx.IsSet("data-ids") {
			return xerrors.Errorf("must provide --data-ids")
		}
		dataIds := cctx.StringSlice("data-ids")
		duration := cctx.Int("duration")
		delay := cctx.Int("delay")
		clientPublish := cctx.Bool("client-publish")

		client, closer, err := getSaoClient(cctx)
		if err != nil {
			return err
		}
		defer closer()

		didManager, signer, err := cliutil.GetDidManager(cctx, client.Cfg)
		if err != nil {
			return err
		}

		proposal := saotypes.RenewProposal{
			Owner:    didManager.Id,
			Duration: uint64(time.Duration(60*60*24*duration) * time.Second / chain.Blocktime),
			Timeout:  int32(delay),
			Data:     dataIds,
		}

		proposalBytes, err := proposal.Marshal()
		if err != nil {
			return err
		}

		jws, err := didManager.CreateJWS(proposalBytes)
		if err != nil {
			return err
		}
		clientProposal := types.OrderRenewProposal{
			Proposal:     proposal,
			JwsSignature: saotypes.JwsSignature(jws.Signatures[0]),
		}

		var results map[string]string
		if clientPublish {
			_, results, err = client.RenewOrder(ctx, signer, clientProposal)
			if err != nil {
				return err
			}
		} else {
			res, err := client.ModelRenewOrder(ctx, &clientProposal, !clientPublish)
			if err != nil {
				return err
			}
			results = res.Results
		}

		var renewModels = make(map[string]uint64, len(results))
		var renewedOrders = make(map[string]string, 0)
		var failedOrders = make(map[string]string, 0)
		for dataId, result := range results {
			if strings.Contains(result, "New order=") {
				orderId, err := strconv.ParseUint(strings.Split(result, "=")[1], 10, 64)
				if err != nil {
					failedOrders[dataId] = result + ", " + err.Error()
				} else {
					renewModels[dataId] = orderId
				}
			} else {
				renewedOrders[dataId] = result
			}
		}

		for dataId, info := range renewedOrders {
			fmt.Printf("successfully renewed model[%s]: %s.\n", dataId, info)
		}

		for dataId, orderId := range renewModels {
			fmt.Printf("successfully renewed model[%s] with orderId[%d].\n", dataId, orderId)
		}

		for dataId, err := range failedOrders {
			fmt.Printf("failed to renew model[%s]: %s.\n", dataId, err)
		}

		return nil
	},
}

var statusCmd = &cli.Command{
	Name:  "status",
	Usage: "check models' status",
	Flags: []cli.Flag{
		&cli.StringSliceFlag{
			Name:     "data-ids",
			Usage:    "data model's dataId list",
			Required: true,
		},
	},
	Action: func(cctx *cli.Context) error {
		ctx := cctx.Context

		if !cctx.IsSet("data-ids") {
			return xerrors.Errorf("must provide --data-ids")
		}
		dataIds := cctx.StringSlice("data-ids")

		client, closer, err := getSaoClient(cctx)
		if err != nil {
			return err
		}
		defer closer()

		didManager, _, err := cliutil.GetDidManager(cctx, client.Cfg)
		if err != nil {
			return err
		}

		gatewayAddress, err := client.GetNodeAddress(ctx)
		if err != nil {
			return err
		}

		states := ""
		for _, dataId := range dataIds {
			proposal := saotypes.QueryProposal{
				Owner:   didManager.Id,
				Keyword: dataId,
			}

			request, err := buildQueryRequest(ctx, didManager, proposal, client, gatewayAddress)
			if err != nil {
				return err
			}

			res, err := client.QueryMetadata(ctx, request, 0)
			if err != nil {
				if len(states) > 0 {
					states = fmt.Sprintf("%s\n[%s]: %s", states, dataId, err.Error())
				} else {
					states = fmt.Sprintf("[%s]: %s", dataId, err.Error())
				}
			} else {
				duration := res.Metadata.Duration
				used := uint64(time.Now().Second()) - res.Metadata.CreatedAt
				if len(states) > 0 {
					states = states + "\n"
				}
				consoleOK := color.New(color.FgGreen, color.Bold)
				consoleWarn := color.New(color.FgHiRed, color.Bold)

				var leftDays uint64
				if duration >= used {
					leftDays = duration - used

				} else {
					leftDays = used - duration
				}
				if leftDays > 5 {
					states = fmt.Sprintf("%s[%s]: expired in %s days", states, dataId, consoleOK.Sprintf("%d", leftDays/(60*60*24*365)))
				} else {
					states = fmt.Sprintf("%s[%s]: expired %s days ago", states, dataId, consoleWarn.Sprintf("%d", leftDays/(60*60*24*365)))
				}
			}
		}

		fmt.Println(states)

		return nil
	},
}

var deleteCmd = &cli.Command{
	Name:  "delete",
	Usage: "delete data model",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:     "data-id",
			Usage:    "data model's dataId",
			Required: true,
		},
	},
	Action: func(cctx *cli.Context) error {
		ctx := cctx.Context

		if !cctx.IsSet("data-id") {
			return xerrors.Errorf("must provide --data-id")
		}
		dataId := cctx.String("data-id")
		clientPublish := cctx.Bool("client-publish")

		client, closer, err := getSaoClient(cctx)
		if err != nil {
			return err
		}
		defer closer()

		didManager, signer, err := cliutil.GetDidManager(cctx, client.Cfg)
		if err != nil {
			return err
		}

		proposal := saotypes.TerminateProposal{
			Owner:  didManager.Id,
			DataId: dataId,
		}

		proposalBytes, err := proposal.Marshal()
		if err != nil {
			return err
		}

		jws, err := didManager.CreateJWS(proposalBytes)
		if err != nil {
			return err
		}
		request := types.OrderTerminateProposal{
			Proposal:     proposal,
			JwsSignature: saotypes.JwsSignature(jws.Signatures[0]),
		}

		if clientPublish {
			_, _, err = client.TerminateOrder(ctx, signer, request)
			if err != nil {
				return err
			}
		}

		result, err := client.ModelDelete(ctx, &request, !clientPublish)
		if err != nil {
			return err
		}

		fmt.Printf("data model %s deleted.\r\n", result.DataId)

		return nil
	},
}

var commitsCmd = &cli.Command{
	Name:  "commits",
	Usage: "list data model historical commits",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:     "keyword",
			Usage:    "data model's alias, dataId or tag",
			Required: true,
		},
	},
	Action: func(cctx *cli.Context) error {
		ctx := cctx.Context

		if !cctx.IsSet("keyword") {
			return xerrors.Errorf("must provide --keyword")
		}
		keyword := cctx.String("keyword")

		client, closer, err := getSaoClient(cctx)
		if err != nil {
			return err
		}
		defer closer()

		didManager, _, err := cliutil.GetDidManager(cctx, client.Cfg)
		if err != nil {
			return err
		}

		groupId := cctx.String("platform")
		if groupId == "" {
			groupId = client.Cfg.GroupId
		}

		proposal := saotypes.QueryProposal{
			Owner:   didManager.Id,
			Keyword: keyword,
			GroupId: groupId,
		}

		if !utils.IsDataId(keyword) {
			proposal.KeywordType = 2
		}

		gatewayAddress, err := client.GetNodeAddress(ctx)
		if err != nil {
			return err
		}

		request, err := buildQueryRequest(ctx, didManager, proposal, client, gatewayAddress)
		if err != nil {
			return err
		}

		resp, err := client.ModelShowCommits(ctx, request)
		if err != nil {
			return err
		}

		console := color.New(color.FgMagenta, color.Bold)

		fmt.Print("  Model DataId : ")
		console.Println(resp.DataId)

		fmt.Print("  Model Alias  : ")
		console.Println(resp.Alias)

		fmt.Println("  -----------------------------------------------------------")
		fmt.Println("  Version |Commit                              |Height")
		fmt.Println("  -----------------------------------------------------------")
		for i, commit := range resp.Commits {
			commitInfo := strings.Split(commit, "\032")
			if len(commitInfo) != 2 || len(commitInfo[1]) == 0 {
				return xerrors.Errorf("invalid commit information: %s", commit)
			}

			console.Printf("  v%d\t  |%s|%s\r\n", i, commitInfo[0], commitInfo[1])
		}
		fmt.Println("  -----------------------------------------------------------")

		return nil
	},
}

var updateCmd = &cli.Command{
	Name:      "update",
	Usage:     "update an existing data model",
	UsageText: "use patch cmd to generate --patch flag and --cid first. permission error will be reported if you don't have model write perm",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:     "patch",
			Usage:    "patch to apply for the data model",
			Required: true,
		},
		&cli.IntFlag{
			Name:     "duration",
			Usage:    "how many days do you want to store the data.",
			Value:    DEFAULT_DURATION,
			Required: false,
		},
		&cli.IntFlag{
			Name:     "delay",
			Usage:    "how many epochs to wait for data update complete",
			Value:    1 * 60,
			Required: false,
		},
		&cli.BoolFlag{
			Name:     "client-publish",
			Usage:    "true if client sends MsgStore message on chain, or leave it to gateway to send",
			Value:    false,
			Required: false,
		},
		&cli.BoolFlag{
			Name:     "force",
			Usage:    "overwrite the latest commit",
			Value:    false,
			Required: false,
		},
		&cli.StringSliceFlag{
			Name:     "tags",
			Required: false,
		},
		&cli.StringFlag{
			Name:     "rule",
			Value:    "",
			Required: false,
		},
		&cli.StringFlag{
			Name:     "keyword",
			Usage:    "data model's alias name, dataId or tag",
			Required: true,
		},
		&cli.StringFlag{
			Name:     "cid",
			Usage:    "target content cid",
			Required: true,
		},
		&cli.IntFlag{
			Name:     "size",
			Usage:    "target content size",
			Required: true,
		},
		&cli.IntFlag{
			Name:     "replica",
			Usage:    "how many copies to store.",
			Value:    DEFAULT_REPLICA,
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
		if !cctx.IsSet("keyword") {
			return xerrors.Errorf("must provide --keyword")
		}
		keyword := cctx.String("keyword")

		size := cctx.Int("size")
		if size <= 0 {
			return xerrors.Errorf("invalid size")
		}

		patch := []byte(cctx.String("patch"))
		contentCid := cctx.String("cid")
		newCid, err := cid.Decode(contentCid)
		if err != nil {
			return err
		}

		extendInfo := cctx.String("extend-info")
		if len(extendInfo) > 1024 {
			return xerrors.Errorf("extend-info should no longer than 1024 characters")
		}

		clientPublish := cctx.Bool("client-publish")

		// TODO: check valid range
		duration := cctx.Int("duration")
		replicas := cctx.Int("replica")
		delay := cctx.Int("delay")
		client, closer, err := getSaoClient(cctx)
		if err != nil {
			return err
		}
		defer closer()

		groupId := cctx.String("platform")
		if groupId == "" {
			groupId = client.Cfg.GroupId
		}

		didManager, signer, err := cliutil.GetDidManager(cctx, client.Cfg)
		if err != nil {
			return err
		}

		gatewayAddress, err := client.GetNodeAddress(ctx)
		if err != nil {
			return err
		}

		queryProposal := saotypes.QueryProposal{
			Owner:   didManager.Id,
			Keyword: keyword,
			GroupId: groupId,
		}

		if !utils.IsDataId(keyword) {
			queryProposal.KeywordType = 2
		}

		request, err := buildQueryRequest(ctx, didManager, queryProposal, client, gatewayAddress)
		if err != nil {
			return err
		}

		res, err := client.QueryMetadata(ctx, request, 0)
		if err != nil {
			return err
		}

		force := cctx.Bool("force")

		operation := uint32(1)

		if force {
			operation = 2
		}

		proposal := saotypes.Proposal{
			Owner:      didManager.Id,
			Provider:   gatewayAddress,
			GroupId:    groupId,
			Duration:   uint64(time.Duration(60*60*24*duration) * time.Second / chain.Blocktime),
			Replica:    int32(replicas),
			Timeout:    int32(delay),
			DataId:     res.Metadata.DataId,
			Alias:      res.Metadata.Alias,
			Tags:       cctx.StringSlice("tags"),
			Cid:        newCid.String(),
			CommitId:   utils.GenerateCommitId(),
			Rule:       cctx.String("rule"),
			Operation:  operation,
			Size_:      uint64(size),
			ExtendInfo: extendInfo,
		}

		clientProposal, err := buildClientProposal(ctx, didManager, proposal, client)
		if err != nil {
			return err
		}

		var orderId uint64 = 0
		if clientPublish {
			resp, _, _, err := client.StoreOrder(ctx, signer, clientProposal)
			if err != nil {
				return err
			}
			orderId = resp.OrderId
		}

		resp, err := client.ModelUpdate(ctx, request, clientProposal, orderId, patch)
		if err != nil {
			return err
		}
		fmt.Printf("alias: %s, data id: %s, commit id: %s.\r\n", resp.Alias, resp.DataId, resp.CommitId)
		return nil
	},
}

var updatePermissionCmd = &cli.Command{
	Name:      "update-permission",
	Usage:     "update data model's permission",
	UsageText: "only data model owner can update permission",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:     "data-id",
			Usage:    "data model's dataId",
			Required: true,
		},
		&cli.StringSliceFlag{
			Name:     "readonly-dids",
			Usage:    "DIDs with read access to the data model",
			Required: false,
		},
		&cli.StringSliceFlag{
			Name:     "readwrite-dids",
			Usage:    "DIDs with read and write access to the data model",
			Required: false,
		},
	},
	Action: func(cctx *cli.Context) error {
		ctx := cctx.Context

		if !cctx.IsSet("data-id") {
			return xerrors.Errorf("must provide --data-id")
		}
		dataId := cctx.String("data-id")
		clientPublish := cctx.Bool("client-publish")

		client, closer, err := getSaoClient(cctx)
		if err != nil {
			return err
		}
		defer closer()

		didManager, signer, err := cliutil.GetDidManager(cctx, client.Cfg)
		if err != nil {
			return err
		}

		proposal := saotypes.PermissionProposal{
			Owner:         didManager.Id,
			DataId:        dataId,
			ReadonlyDids:  cctx.StringSlice("readonly-dids"),
			ReadwriteDids: cctx.StringSlice("readwrite-dids"),
		}

		proposalBytes, err := proposal.Marshal()
		if err != nil {
			return err
		}

		jws, err := didManager.CreateJWS(proposalBytes)
		if err != nil {
			return err
		}

		request := &types.PermissionProposal{
			Proposal: proposal,
			JwsSignature: saotypes.JwsSignature{
				Protected: jws.Signatures[0].Protected,
				Signature: jws.Signatures[0].Signature,
			},
		}

		if clientPublish {
			_, err = client.UpdatePermission(ctx, signer, request)
			if err != nil {
				return err
			}
		} else {
			_, err = client.ModelUpdatePermission(ctx, request, !clientPublish)
			if err != nil {
				return err
			}
		}

		fmt.Printf("Data model[%s]'s permission updated.\r\n", dataId)
		return nil
	},
}

var patchGenCmd = &cli.Command{
	Name:      "patch-gen",
	Usage:     "generate data model patch",
	UsageText: "used to before update cmd. you will get patch diff and target cid.",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:     "origin",
			Usage:    "the original data model content",
			Required: true,
		},
		&cli.StringFlag{
			Name:     "target",
			Usage:    "the target data model content",
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

		var newModel interface{}
		err = json.Unmarshal(content, &newModel)
		if err != nil {
			return err
		}

		var targetModel interface{}
		err = json.Unmarshal([]byte(target), &targetModel)
		if err != nil {
			return err
		}

		valueStrNew, err := json.Marshal(newModel)
		if err != nil {
			return err
		}

		valueStrTarget, err := json.Marshal(targetModel)
		if err != nil {
			return err
		}

		if string(valueStrNew) != string(valueStrTarget) {
			return xerrors.Errorf("failed to generate the patch")
		}

		targetCid, err := utils.CalculateCid(content)
		if err != nil {
			return err
		}

		console := color.New(color.FgMagenta, color.Bold)

		fmt.Print("  Patch      : ")
		console.Println(patch)

		fmt.Print("  Target Cid : ")
		console.Println(targetCid)

		fmt.Print("  Target Size : ")
		console.Println(len(content))

		return nil
	},
}

func buildClientProposal(_ context.Context, didManager *did.DidManager, proposal saotypes.Proposal, _ chain.ChainSvcApi) (*types.OrderStoreProposal, error) {
	if proposal.Owner == "all" {
		return &types.OrderStoreProposal{
			Proposal: proposal,
		}, nil
	}

	proposalBytes, err := proposal.Marshal()
	if err != nil {
		return nil, err
	}

	jws, err := didManager.CreateJWS(proposalBytes)
	if err != nil {
		return nil, err
	}
	return &types.OrderStoreProposal{
		Proposal: proposal,
		JwsSignature: saotypes.JwsSignature{
			Protected: jws.Signatures[0].Protected,
			Signature: jws.Signatures[0].Signature,
		},
	}, nil
}

func buildQueryRequest(ctx context.Context, didManager *did.DidManager, proposal saotypes.QueryProposal, chain chain.ChainSvcApi, gatewayAddress string) (*types.MetadataProposal, error) {
	lastHeight, err := chain.GetLastHeight(ctx)
	if err != nil {
		return nil, err
	}

	peerInfo, err := chain.GetNodePeer(ctx, gatewayAddress)
	if err != nil {
		return nil, err
	}

	proposal.LastValidHeight = uint64(lastHeight + 200)
	proposal.Gateway = peerInfo

	if proposal.Owner == "all" {
		return &types.MetadataProposal{
			Proposal: proposal,
		}, nil
	}

	proposalBytes, err := proposal.Marshal()
	if err != nil {
		return nil, err
	}

	jws, err := didManager.CreateJWS(proposalBytes)
	if err != nil {
		return nil, err
	}

	return &types.MetadataProposal{
		Proposal: proposal,
		JwsSignature: saotypes.JwsSignature{
			Protected: jws.Signatures[0].Protected,
			Signature: jws.Signatures[0].Signature,
		},
	}, nil
}
