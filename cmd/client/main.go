package main

// TODO:
// * how to generate cid from scratch
// * guic transfer data

import (
	"bufio"
	"cosmossdk.io/math"
	"fmt"
	logging "github.com/ipfs/go-log/v2"
	"github.com/urfave/cli/v2"
	"os"
	"sao-storage-node/build"
	"sao-storage-node/chain"
	"sao-storage-node/client"
	cliutil "sao-storage-node/cmd"
	"sao-storage-node/cmd/account"
	"strings"
)

const (
	DEFAULT_DURATION = 365
	DEFAULT_REPLICA  = 1

	FlagClientRepo = "repo"
	FlagGateway    = "gateway"
	FlagKeyName    = "keyName"
)

var flagRepo = &cli.StringFlag{
	Name:     FlagClientRepo,
	Usage:    "repo directory for sao client",
	Required: false,
	EnvVars:  []string{"SAO_CLIENT_PATH"},
	Value:    "~/.sao-cli",
}

var flagGateway = &cli.StringFlag{
	Name:     FlagGateway,
	EnvVars:  []string{"SAO_GATEWAY_API"},
	Required: false,
}

var flagPlatform = &cli.StringFlag{
	Name:     "platform",
	Usage:    "platform to manage the data model",
	Required: false,
}

func getSaoClient(cctx *cli.Context) (*client.SaoClient, func(), error) {
	opt := client.SaoClientOptions{
		Repo:      cctx.String(FlagClientRepo),
		Gateway:   cctx.String(FlagGateway),
		ChainAddr: cliutil.ChainAddress,
		KeyName:   cctx.String(FlagKeyName),
	}
	return client.NewSaoClient(cctx.Context, opt)
}

func before(cctx *cli.Context) error {
	// by default, do not print any log for client.
	_ = logging.SetLogLevel("saoclient", "TRACE")
	_ = logging.SetLogLevel("chain", "TRACE")
	_ = logging.SetLogLevel("transport-client", "TRACE")

	if cliutil.IsVeryVerbose {
		_ = logging.SetLogLevel("saoclient", "DEBUG")
		_ = logging.SetLogLevel("chain", "DEBUG")
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
			cliutil.FlagChainAddress,
			flagRepo,
			flagGateway,
			flagPlatform,
			cliutil.FlagNetType,
			cliutil.FlagVeryVerbose,
		},
		Commands: []*cli.Command{
			initCmd,
			account.AccountCmd,
			modelCmd,
			fileCmd,
			didCmd,
		},
	}
	app.Setup()

	if err := app.Run(os.Args); err != nil {
		os.Stderr.WriteString("Error: " + err.Error() + "\n")
		os.Exit(1)
	}
}

var initCmd = &cli.Command{
	Name: "init",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:     FlagKeyName,
			Usage:    "cosmos account key name",
			Required: true,
		},
	},
	Action: func(cctx *cli.Context) error {
		repo := cctx.String(FlagClientRepo)

		saoclient, closer, err := getSaoClient(cctx)
		if err != nil {
			return err
		}
		defer closer()
		fmt.Printf("repo %s is initialized.", repo)
		fmt.Println()

		accountName, address, mnemonic, err := chain.Create(cctx.Context, repo, saoclient.Cfg.KeyName)
		fmt.Println("account created: ")
		fmt.Println("Account:", accountName)
		fmt.Println("Address:", address)
		fmt.Println("Mnemonic:", mnemonic)

		for {
			coins, err := saoclient.GetBalance(cctx.Context, address)
			askFor := false
			if err != nil {
				fmt.Errorf("%v", err)
				askFor = true
			} else {
				if coins.AmountOf("stake").LT(math.NewInt(1000)) {
					askFor = true
				}
			}
			if askFor {
				fmt.Print("Please deposit enough coins to pay gas. Confirm with 'yes' :")
				reader := bufio.NewReader(os.Stdin)
				indata, err := reader.ReadBytes('\n')
				if err != nil {
					return err
				}
				_ = strings.Replace(string(indata), "\n", "", -1)
			} else {
				break
			}
		}

		didManager, address, err := cliutil.GetDidManager(cctx, saoclient.Cfg)
		if err != nil {
			return err
		}

		hash, err := saoclient.UpdateDidBinding(cctx.Context, address, didManager.Id, fmt.Sprintf("cosmos:sao:%s", address))
		if err != nil {
			return err
		}

		err = saoclient.SaveConfig(saoclient.Cfg)
		if err != nil {
			return fmt.Errorf("save local config failed: %v", err)
		}

		fmt.Printf("Created DID %s. tx hash %s", didManager.Id, hash)
		fmt.Println()
		fmt.Println("sao client initialized.")
		return nil
	},
}
