package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/SaoNetwork/sao-node/chain"
	saoclient "github.com/SaoNetwork/sao-node/client"
	cliutil "github.com/SaoNetwork/sao-node/cmd"
	"github.com/SaoNetwork/sao-node/types"
	"github.com/SaoNetwork/sao-node/utils"

	saotypes "github.com/SaoNetwork/sao/x/sao/types"
	"github.com/fatih/color"
	"github.com/ipfs/go-cid"
	"github.com/urfave/cli/v2"
)

var fileCmd = &cli.Command{
	Name:  "file",
	Usage: "file management",
	Subcommands: []*cli.Command{
		createFileCmd,
		uploadCmd,
		downloadCmd,
	},
}

var createFileCmd = &cli.Command{
	Name:  "create",
	Usage: "ModelCreate a file",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:     "file-name",
			Usage:    "local file path",
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
			Usage:    "how many epochs to wait for the file ready",
			Value:    1 * 60,
			Required: false,
		},
		&cli.BoolFlag{
			Name:     "client-publish",
			Usage:    "true if client sends MsgStore message on chain, or leave it to gateway to send",
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
			Name:     "cid",
			Value:    "",
			Required: true,
		},
		&cli.IntFlag{
			Name:     "size",
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
			Value:    "",
			Required: false,
		},
	},
	Action: func(cctx *cli.Context) error {
		ctx := cctx.Context

		// ---- check parameters ----
		if !cctx.IsSet("file-name") {
			return types.Wrapf(types.ErrInvalidParameters, "must provide --file-name")
		}
		fileName := types.Type_Prefix_File + cctx.String("file-name")

		clientPublish := cctx.Bool("client-publish")

		// TODO: check valid range
		duration := cctx.Int("duration")
		replicas := cctx.Int("replica")
		delay := cctx.Int("delay")
		size := cctx.Uint64("size")

		extendInfo := cctx.String("extend-info")
		if len(extendInfo) > 1024 {
			return types.Wrapf(types.ErrInvalidParameters, "extend-info should no longer than 1024 characters")
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

		contentCid, err := cid.Decode(cctx.String("cid"))
		if err != nil {
			return types.Wrap(types.ErrInvalidCid, err)
		}

		didManager, signer, err := cliutil.GetDidManager(cctx, client.Cfg.KeyName)
		if err != nil {
			return err
		}

		gatewayAddress, err := client.GetNodeAddress(ctx)
		if err != nil {
			return err
		}

		dataId := utils.GenerateDataId(didManager.Id + groupId)
		proposal := saotypes.Proposal{
			DataId:     dataId,
			Owner:      didManager.Id,
			Provider:   gatewayAddress,
			GroupId:    groupId,
			Duration:   uint64(time.Duration(60*60*24*duration) * time.Second / chain.Blocktime),
			Replica:    int32(replicas),
			Timeout:    int32(delay),
			Alias:      fileName,
			Tags:       cctx.StringSlice("tags"),
			Cid:        contentCid.String(),
			CommitId:   dataId,
			Rule:       cctx.String("rule"),
			Operation:  1,
			ExtendInfo: extendInfo,
			Size_:      size,
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

		queryProposal := saotypes.QueryProposal{
			Owner:   didManager.Id,
			Keyword: dataId,
		}

		request, err := buildQueryRequest(ctx, didManager, queryProposal, client, gatewayAddress)
		if err != nil {
			return err
		}

		resp, err := client.ModelCreateFile(ctx, request, clientProposal, orderId)
		if err != nil {
			return err
		}
		fmt.Printf("file name: %s, data id: %s\r\n", resp.Alias, resp.DataId)
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
		&cli.StringFlag{
			Name:  "protocol",
			Usage: "protocol to use (tcp/udp)",
			Value: "udp",
		},
	},
	Action: func(cctx *cli.Context) error {
		ctx := cctx.Context

		fpath := cctx.String("filepath")
		multiaddr := cctx.String("multiaddr")
		protocol := cctx.String("protocol")

		if !strings.Contains(multiaddr, "/p2p/") {
			return types.Wrapf(types.ErrInvalidParameters, "invalid multiaddr: %s", multiaddr)
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
				fmt.Printf("skip directory %s\r\n", path)
			}

			return nil
		})

		if err != nil {
			return types.Wrap(types.ErrInvalidParameters, err)
		}

		repo := cctx.String(FlagClientRepo)
		for _, file := range files {
			var c cid.Cid
			if protocol == "tcp" {
				c = saoclient.DoTransportTCP(ctx, repo, multiaddr, file)
			} else {
				c = saoclient.DoTransport(ctx, repo, multiaddr, peerId, file)
			}

			if c != cid.Undef {
				fmt.Printf("file [%s] successfully uploaded, CID is %s.\r\n", file, c.String())
			} else {
				fmt.Printf("failed to uploaded the file [%s], please try again", file)
			}
		}

		return nil
	},
}

var downloadCmd = &cli.Command{
	Name:  "download",
	Usage: "download file(s) from storage network",
	Flags: []cli.Flag{
		&cli.StringSliceFlag{
			Name:     "keywords",
			Usage:    "storage network dataId(s) of the file(s)",
			Required: true,
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
	},
	Action: func(cctx *cli.Context) error {
		ctx := cctx.Context

		if !cctx.IsSet("keywords") {
			return types.Wrapf(types.ErrInvalidParameters, "must provide --keywords")
		}
		keywords := cctx.StringSlice("keywords")

		client, closer, err := getSaoClient(cctx)
		if err != nil {
			return err
		}
		defer closer()

		groupId := cctx.String("platform")
		if groupId == "" {
			groupId = client.Cfg.GroupId
		}

		version := cctx.String("version")
		commitId := cctx.String("commit-id")
		if cctx.IsSet("version") && cctx.IsSet("commit-id") {
			fmt.Println("--version is to be ignored once --commit-id is specified")
			version = ""
		}

		didManager, _, err := cliutil.GetDidManager(cctx, client.Cfg.KeyName)
		if err != nil {
			return err
		}

		gatewayAddress, err := client.GetNodeAddress(ctx)
		if err != nil {
			return err
		}

		for _, keyword := range keywords {
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

			request, err := buildQueryRequest(ctx, didManager, proposal, client, gatewayAddress)
			if err != nil {
				return err
			}

			resp, err := client.ModelLoad(ctx, request)
			if err != nil {
				return err
			}

			console := color.New(color.FgMagenta, color.Bold)

			fmt.Print("  File DataId   : ")
			console.Println(resp.DataId)

			fmt.Print("  File Name     : ")
			console.Println(resp.Alias)

			fmt.Print("  File CommitId : ")
			console.Println(resp.CommitId)

			fmt.Print("  File Version  : ")
			console.Println(resp.Version)

			fmt.Print("  File Cid      : ")
			console.Println(resp.Cid)

			path := filepath.Join("./", resp.Alias)
			dir, _ := filepath.Split(path)
			os.MkdirAll(dir, 0775)
			file, err := os.Create(path)
			if err != nil {
				return err
			}

			_, err = file.Write(resp.Content)
			if err != nil {
				return err
			}
			fmt.Printf("file downloaded to %s\r\n", path)
		}

		return nil
	},
}
