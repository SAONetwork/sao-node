package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sao-storage-node/chain"
	saoclient "sao-storage-node/client"
	cliutil "sao-storage-node/cmd"
	"sao-storage-node/types"
	"sao-storage-node/utils"
	"strings"
	"time"

	saotypes "github.com/SaoNetwork/sao/x/sao/types"
	"github.com/fatih/color"
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
		tokenGenCmd,
		uploadCmd,
		downloadCmd,
	},
}

var createFileCmd = &cli.Command{
	Name: "create-file",
	Flags: []cli.Flag{
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
		&cli.IntFlag{
			Name:     "delay",
			Usage:    "how long to wait for the file ready",
			Value:    24 * 60 * 60,
			Required: false,
		},
		&cli.BoolFlag{
			Name:     "client-publish",
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
		// if !cctx.IsSet("content") || cctx.String("content") == "" {
		// 	return xerrors.Errorf("must provide non-empty --content.")
		// }

		if !cctx.IsSet("file-name") {
			return xerrors.Errorf("must provide --file-name")
		}
		fileName := types.Type_Prefix_File + cctx.String("file-name")

		clientPublish := cctx.Bool("client-publish")

		// TODO: check valid range
		duration := cctx.Int("duration")
		replicas := cctx.Int("replica")
		delay := cctx.Int("delay")

		extendInfo := cctx.String("extend-info")
		if len(extendInfo) > 1024 {
			return xerrors.Errorf("extend-info should no longer than 1024 characters")
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
			return err
		}

		didManager, signer, err := cliutil.GetDidManager(cctx, client.Cfg)
		if err != nil {
			return err
		}

		gatewayAddress, err := client.NodeAddress(ctx)
		if err != nil {
			return err
		}

		dataId := utils.GenerateDataId()
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
			Operation:  0,
			ExtendInfo: extendInfo,
		}

		clientProposal, err := buildClientProposal(ctx, didManager, proposal, client)
		if err != nil {
			return err
		}

		var orderId uint64 = 0
		if clientPublish {
			orderId, _, err = client.StoreOrder(ctx, signer, clientProposal)
			if err != nil {
				return err
			}
		}

		queryProposal := saotypes.QueryProposal{
			Owner:   didManager.Id,
			Keyword: dataId,
		}

		request, err := buildQueryRequest(ctx, didManager, queryProposal, client, gatewayAddress)
		if err != nil {
			return err
		}

		resp, err := client.CreateFile(ctx, request, clientProposal, orderId)
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
				fmt.Printf("skip directory %s\r\n", path)
			}

			return nil
		})

		if err != nil {
			return err
		}

		repo := cctx.String(FlagClientRepo)
		for _, file := range files {
			c := saoclient.DoTransport(ctx, repo, multiaddr, peerId, file)
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
			return xerrors.Errorf("must provide --keywords")
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

		didManager, _, err := cliutil.GetDidManager(cctx, client.Cfg)
		if err != nil {
			return err
		}

		gatewayAddress, err := client.NodeAddress(ctx)
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
				proposal.Type_ = 2
			}

			request, err := buildQueryRequest(ctx, didManager, proposal, client, gatewayAddress)
			if err != nil {
				return err
			}

			resp, err := client.Load(ctx, request)
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
			file, err := os.Create(path)
			if err != nil {
				return err
			}

			_, err = file.Write([]byte(resp.Content))
			if err != nil {
				return err
			}
			fmt.Printf("file downloaded to %s\r\n", path)
		}

		return nil
	},
}

var peerInfoCmd = &cli.Command{
	Name:  "peer-info",
	Usage: "get peer info of the gateway",
	Action: func(cctx *cli.Context) error {
		ctx := cctx.Context

		gateway := cctx.String(FlagGateway)
		client, closer, err := getSaoClient(cctx)
		if err != nil {
			return err
		}
		defer closer()

		resp, err := client.GetPeerInfo(ctx)
		if err != nil {
			return err
		}

		console := color.New(color.FgMagenta, color.Bold)

		fmt.Print("  GateWay   : ")
		console.Println(gateway)

		fmt.Print("  Peer Info : ")
		console.Println(resp.PeerInfo)

		return nil
	},
}

var tokenGenCmd = &cli.Command{
	Name:  "token-gen",
	Usage: "generate token to access http file server",
	Action: func(cctx *cli.Context) error {
		ctx := cctx.Context

		gateway := cctx.String(FlagGateway)
		client, closer, err := getSaoClient(cctx)
		if err != nil {
			return err
		}
		defer closer()

		didManager, _, err := cliutil.GetDidManager(cctx, client.Cfg)
		if err != nil {
			return err
		}

		resp, err := client.GenerateToken(ctx, didManager.Id)
		if err != nil {
			return err
		}

		console := color.New(color.FgMagenta, color.Bold)

		fmt.Print("  DID     : ")
		console.Println(didManager.Id)

		fmt.Print("  GateWay : ")
		console.Println(gateway)

		fmt.Print("  Server  : ")
		console.Println(resp.Server)

		fmt.Print("  Token   : ")
		console.Println(resp.Token)

		return nil
	},
}
