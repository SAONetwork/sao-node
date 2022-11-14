package main

import (
	"fmt"
	"os"
	"path/filepath"
	apiclient "sao-storage-node/api/client"
	apitypes "sao-storage-node/api/types"
	"sao-storage-node/chain"
	saoclient "sao-storage-node/client"
	cliutil "sao-storage-node/cmd"
	"sao-storage-node/types"
	"sao-storage-node/utils"
	"strings"

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
			Name:     "secret",
			Usage:    "client secret",
			Required: false,
		},
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
		&cli.IntFlag{
			Name:     "delay",
			Usage:    "how long to wait for the file ready",
			Value:    24 * 60 * 60,
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
		fileName := types.Type_Prefix_File + cctx.String("file-name")

		clientPublish := cctx.Bool("clientPublish")

		// TODO: check valid range
		duration := cctx.Int("duration")
		replicas := cctx.Int("replica")
		delay := cctx.Int("delay")
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

		contentCid, err := cid.Decode(cctx.String("cid"))
		if err != nil {
			return err
		}

		didManager, err := cliutil.GetDidManager(cctx, client.Cfg.Seed, client.Cfg.Alg)
		if err != nil {
			return err
		}

		gatewayAddress, err := gatewayApi.NodeAddress(ctx)
		if err != nil {
			return err
		}

		dataId := utils.GenerateDataId()
		proposal := saotypes.Proposal{
			DataId:     dataId,
			Owner:      didManager.Id,
			Provider:   gatewayAddress,
			GroupId:    groupId,
			Duration:   int32(duration),
			Replica:    int32(replicas),
			Timeout:    int32(delay),
			Alias:      fileName,
			Tags:       cctx.StringSlice("tags"),
			Cid:        contentCid.String(),
			CommitId:   dataId,
			Rule:       cctx.String("rule"),
			IsUpdate:   false,
			ExtendInfo: extendInfo,
		}

		proposalJsonBytes, err := proposal.Marshal()
		if err != nil {
			return err
		}
		jws, err := didManager.CreateJWS(proposalJsonBytes)
		if err != nil {
			return err
		}
		clientProposal := types.ClientOrderProposal{
			Proposal:        proposal,
			ClientSignature: jws.Signatures[0],
		}

		var orderId uint64 = 0
		if clientPublish {
			chain, err := chain.NewChainSvc(ctx, "cosmos", chainAddress, "/websocket")
			if err != nil {
				return xerrors.Errorf("new cosmos chain: %w", err)
			}

			orderId, _, err = chain.StoreOrder(ctx, owner, clientProposal)
			if err != nil {
				return err
			}
		}

		resp, err := client.CreateFile(ctx, clientProposal, orderId)
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

		for _, file := range files {
			c := saoclient.DoTransport(ctx, multiaddr, peerId, file)
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
		&cli.StringFlag{
			Name:     "secret",
			Usage:    "client secret",
			Required: false,
		},
		&cli.StringSliceFlag{
			Name:     "keywords",
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

		if !cctx.IsSet("keywords") {
			return xerrors.Errorf("must provide --keywords")
		}
		keywords := cctx.StringSlice("keywords")

		client := saoclient.NewSaoClient(gatewayApi)

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

		didManager, err := cliutil.GetDidManager(cctx, client.Cfg.Seed, client.Cfg.Alg)
		if err != nil {
			return err
		}

		for _, keyword := range keywords {
			req := apitypes.LoadReq{
				KeyWord:   keyword,
				PublicKey: didManager.Id,
				GroupId:   groupId,
				CommitId:  commitId,
				Version:   version,
			}

			resp, err := client.Load(ctx, req)
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
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:     "secret",
			Usage:    "client secret",
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

		client := saoclient.NewSaoClient(gatewayApi)

		didManager, err := cliutil.GetDidManager(cctx, client.Cfg.Seed, client.Cfg.Alg)
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
