package main

// TODO:
// * how to generate cid from scratch
// * guic transfer data

import (
	"fmt"
	logging "github.com/ipfs/go-log/v2"
	"github.com/urfave/cli/v2"
	"golang.org/x/xerrors"
	"os"
	apiclient "sao-storage-node/api/client"
	"sao-storage-node/build"
	saoclient "sao-storage-node/client"
	cliutil "sao-storage-node/cmd"
	"sao-storage-node/types"
)

var log = logging.Logger("saoclient")

const (
	DEFAULT_DURATION = 365
	DEFAULT_REPLICA  = 1
)

func before(cctx *cli.Context) error {
	_ = logging.SetLogLevel("saoclient", "INFO")

	if cliutil.IsVeryVerbose {
		_ = logging.SetLogLevel("saoclient", "DEBUG")
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
		},
	}
	app.Setup()

	if err := app.Run(os.Args); err != nil {
		os.Stderr.WriteString("Error: " + err.Error() + "\n")
		os.Exit(1)
	}
}

var testCmd = &cli.Command{
	Name: "create",
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
		fmt.Println(resp)
		return nil
	},
}

var createCmd = &cli.Command{
	Name: "create",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:     "from",
			Required: true,
		},
		&cli.StringFlag{
			Name:     "content",
			Required: true,
		},
		&cli.IntFlag{
			Name:     "duration",
			Usage:    "how long do you want to store the data.",
			Value:    DEFAULT_DURATION,
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
	},
	Action: func(cctx *cli.Context) error {
		ctx := cctx.Context

		// ---- check parameters ----
		if !cctx.IsSet("content") || cctx.String("content") == "" {
			return xerrors.Errorf("must provide non-empty --content.")
		}
		content := cctx.String("content")

		if !cctx.IsSet("from") {
			return xerrors.Errorf("must provide --from")
		}
		from := cctx.String("from")

		// TODO: check valid range
		duration := cctx.Int("duration")
		replicas := cctx.Int("replicas")

		gateway := cctx.String("gateway")
		gatewayApi, closer, err := apiclient.NewGatewayApi(ctx, gateway, nil)
		if err != nil {
			return err
		}
		defer closer()

		client := saoclient.NewSaoClient(gatewayApi)
		dataId, err := client.Create(ctx, types.OrderMeta{
			Creator:  from,
			Duration: duration,
			Replica:  replicas,
		}, content)
		if err != nil {
			return err
		}
		fmt.Println(dataId)
		return nil
	},
}

//var storeCmd = &cli.Command{
//	Name:  "order",
//	Usage: "submit a order",
//	Flags: []cli.Flag{
//		&cli.StringFlag{
//			Name:     "from",
//			Usage:    "client address",
//			Required: true,
//		},
//		&cli.StringFlag{
//			Name:     "content",
//			Usage:    "data content to store",
//			Required: false,
//		},
//		&cli.PathFlag{
//			Name:     "filepath",
//			Usage:    "file's path to store. if --content is provided, --filepath will be ignored",
//			Required: false,
//		},
//		&cli.IntFlag{
//			Name:  "duration",
//			Usage: "how long do you want to store the data.",
//			//Value:    DEFAULT_DURATION,
//			Required: false,
//		},
//		&cli.IntFlag{
//			Name:  "replica",
//			Usage: "how many copies to store.",
//			//Value:    DEFAULT_REPLICA,
//			Required: false,
//		},
//		&cli.StringSliceFlag{
//			Name:     "gateways",
//			Usage:    "gateway connection list, separated by comma",
//			Required: false,
//		},
//	},
//	Action: func(cctx *cli.Context) error {
//		var dataReader io.Reader
//		var err error
//		if cctx.IsSet("content") {
//			dataReader = strings.NewReader(cctx.String("content"))
//		} else if cctx.IsSet("filepath") {
//			f, err := os.Open(cctx.String("filepath"))
//			if err != nil {
//				return err
//			}
//			dataReader = f
//			defer f.Close()
//		} else {
//			return xerrors.Errorf("either --content or --filepath should be provided.")
//		}
//
//		// calculate data cid
//		cid, err := generateDataCid(dataReader)
//		if err != nil {
//			return err
//		}
//
//		// gateway selection.
//		var gateways []string
//		if cctx.IsSet("gateways") {
//			gateways = strings.Split(cctx.String("gateways"), ",")
//		} else {
//			// TODO: should configure default gateway or read from env.
//			//return xerrors.Errorf("--gateways must be provided.")
//		}
//
//		nodeInfo, err := uploadData(gateways, dataReader)
//		if err != nil {
//			return err
//		}
//
//		// store message on chain
//		addressPrefix := "cosmos"
//		cosmos, err := cosmosclient.New(cctx.Context, cosmosclient.WithAddressPrefix(addressPrefix))
//		if err != nil {
//			return err
//		}
//
//		account, err := cosmos.Account(cctx.String("from"))
//		if err != nil {
//			return err
//		}
//
//		addr, err := account.Address(addressPrefix)
//		if err != nil {
//			return err
//		}
//
//		msg := &types.MsgStore{
//			Creator:  addr,
//			Cid:      cid.String(),
//			Provider: nodeInfo.Address,
//			Duration: int32(cctx.Int("duration")),
//			Replica:  int32(cctx.Int("replica")),
//		}
//		txResp, err := cosmos.BroadcastTx(account, msg)
//		if err != nil {
//			return err
//		}
//		log.Debug("MsgStore result: ", txResp)
//		if txResp.TxResponse.Code != 0 {
//			return xerrors.Errorf("MsgStore transaction %v failed: code=%d", txResp.TxResponse.TxHash, txResp.TxResponse.Code)
//		} else {
//			dataResp := &types.MsgStoreResponse{}
//			err = txResp.Decode(dataResp)
//			if err != nil {
//				return err
//			}
//			log.Infof("MsgStore transaction %v succeed: orderId=%d", txResp.TxResponse.TxHash, dataResp.OrderId)
//		}
//
//		return nil
//	},
//}
