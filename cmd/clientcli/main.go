package main

// TODO:
// * how to generate cid from scratch
// * guic transfer data

import (
	"github.com/SaoNetwork/sao/x/sao/types"
	"github.com/ignite/cli/ignite/pkg/cosmosclient"
	"github.com/ipfs/go-cid"
	logging "github.com/ipfs/go-log/v2"
	"github.com/urfave/cli/v2"
	"golang.org/x/xerrors"
	"io"
	"os"
	"sao-storage-node/build"
	client_sdk "sao-storage-node/client"
	cliutil "sao-storage-node/cmd"
	"strings"
)

var log = logging.Logger("clientcli")

const (
	DEFAULT_DURATION = 365
	DEFAULT_REPLICA  = 3
)

func before(cctx *cli.Context) error {
	_ = logging.SetLogLevel("clientcli", "INFO")

	if cliutil.IsVeryVerbose {
		_ = logging.SetLogLevel("clientcli", "DEBUG")
	}

	return nil
}

func main() {
	app := &cli.App{
		Name:                 "clientcli",
		Usage:                "cli client for network client",
		EnableBashCompletion: true,
		Version:              build.UserVersion(),
		Before:               before,
		Flags: []cli.Flag{
			cliutil.FlagVeryVerbose,
		},
		Commands: []*cli.Command{
			storeCmd,
		},
	}
	app.Setup()

	if err := app.Run(os.Args); err != nil {
		os.Stderr.WriteString("Error: " + err.Error() + "\n")
		os.Exit(1)
	}
}

var storeCmd = &cli.Command{
	Name:  "order",
	Usage: "submit a order",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:     "from",
			Usage:    "client address",
			Required: true,
		},
		&cli.StringFlag{
			Name:     "content",
			Usage:    "data content to store",
			Required: false,
		},
		&cli.PathFlag{
			Name:     "filepath",
			Usage:    "file's path to store. if --content is provided, --filepath will be ignored",
			Required: false,
		},
		&cli.IntFlag{
			Name:  "duration",
			Usage: "how long do you want to store the data.",
			//Value:    DEFAULT_DURATION,
			Required: false,
		},
		&cli.IntFlag{
			Name:  "replica",
			Usage: "how many copies to store.",
			//Value:    DEFAULT_REPLICA,
			Required: false,
		},
		&cli.StringSliceFlag{
			Name:     "gateways",
			Usage:    "gateway connection list, separated by comma",
			Required: false,
		},
	},
	Action: func(cctx *cli.Context) error {
		var dataReader io.Reader
		var err error
		if cctx.IsSet("content") {
			dataReader = strings.NewReader(cctx.String("content"))
		} else if cctx.IsSet("filepath") {
			f, err := os.Open(cctx.String("filepath"))
			if err != nil {
				return err
			}
			dataReader = f
			defer f.Close()
		} else {
			return xerrors.Errorf("either --content or --filepath should be provided.")
		}

		// calculate data cid
		cid, err := generateDataCid(dataReader)
		if err != nil {
			return err
		}

		// gateway selection.
		var gateways []string
		if cctx.IsSet("gateways") {
			gateways = strings.Split(cctx.String("gateways"), ",")
		} else {
			// TODO: should configure default gateway or read from env.
			//return xerrors.Errorf("--gateways must be provided.")
		}

		nodeInfo, err := uploadData(gateways, dataReader)
		if err != nil {
			return err
		}

		// store message on chain
		addressPrefix := "cosmos"
		cosmos, err := cosmosclient.New(cctx.Context, cosmosclient.WithAddressPrefix(addressPrefix))
		if err != nil {
			return err
		}

		account, err := cosmos.Account(cctx.String("from"))
		if err != nil {
			return err
		}

		addr, err := account.Address(addressPrefix)
		if err != nil {
			return err
		}

		msg := &types.MsgStore{
			Creator:  addr,
			Cid:      cid.String(),
			Provider: nodeInfo.Address,
			Duration: int32(cctx.Int("duration")),
			Replica:  int32(cctx.Int("replica")),
		}
		txResp, err := cosmos.BroadcastTx(account, msg)
		if err != nil {
			return err
		}
		log.Debug("MsgStore result: ", txResp)
		if txResp.TxResponse.Code != 0 {
			return xerrors.Errorf("MsgStore transaction %v failed: code=%d", txResp.TxResponse.TxHash, txResp.TxResponse.Code)
		} else {
			dataResp := &types.MsgStoreResponse{}
			err = txResp.Decode(dataResp)
			if err != nil {
				return err
			}
			log.Infof("MsgStore transaction %v succeed: orderId=%d", txResp.TxResponse.TxHash, dataResp.OrderId)
		}

		return nil
	},
}

// upload data from reader to a proper gateway.
func uploadData(gatewayList []string, reader io.Reader) (client_sdk.GatewayNodeInfo, error) {
	// TODO: transfer file to gateway node.
	return client_sdk.GatewayNodeInfo{
		Address: "cosmos1qg9l86zta6kyrajlhh48raljdzedextkvsjcjh",
	}, nil
}

func generateDataCid(reader io.Reader) (cid.Cid, error) {
	// TODO:
	return cid.Decode("QmeSoArjthZ5VcaeJxg35rRPt6gwd4sWyPmNbYSpKtF4uF")
}
