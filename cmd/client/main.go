package main

// TODO:
// * how to generate cid from scratch
// * guic transfer data

import (
	"bufio"
	"fmt"
	"os"
	"sao-node/build"
	"sao-node/chain"
	"sao-node/client"
	cliutil "sao-node/cmd"
	"sao-node/cmd/account"
	"sao-node/types"
	"strings"

	"cosmossdk.io/math"
	logging "github.com/ipfs/go-log/v2"
	"github.com/urfave/cli/v2"
)

const (
	DEFAULT_DURATION = 365
	DEFAULT_REPLICA  = 1

	FlagClientRepo = "repo"
)

var flagRepo = &cli.StringFlag{
	Name:     FlagClientRepo,
	Usage:    "repo directory for sao client",
	Required: false,
	EnvVars:  []string{"SAO_CLIENT_PATH"},
	Value:    "~/.sao-cli",
}

var flagPlatform = &cli.StringFlag{
	Name:     "platform",
	Usage:    "platform to manage the data model",
	Required: false,
}

func getSaoClient(cctx *cli.Context) (*client.SaoClient, func(), error) {
	opt := client.SaoClientOptions{
		Repo:        cctx.String(FlagClientRepo),
		Gateway:     cliutil.Gateway,
		ChainAddr:   cliutil.ChainAddress,
		KeyName:     cctx.String(cliutil.FlagKeyName),
		KeyringHome: cliutil.KeyringHome,
	}
	return client.NewSaoClient(cctx.Context, opt)
}

func before(_ *cli.Context) error {
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
		Name:                 cliutil.APP_NAME_CLIENT,
		Usage:                "command line for sao network client",
		EnableBashCompletion: true,
		Version:              build.UserVersion(),
		Before:               before,
		Flags: []cli.Flag{
			cliutil.FlagChainAddress,
			flagRepo,
			cliutil.FlagGateway,
			flagPlatform,
			cliutil.FlagVeryVerbose,
			cliutil.FlagKeyringHome,
		},
		Commands: []*cli.Command{
			initCmd,
			netCmd,
			modelCmd,
			fileCmd,
			didCmd,
			account.AccountCmd,
			cliutil.GenerateDocCmd,
		},
	}
	app.Setup()

	if err := app.Run(os.Args); err != nil {
		os.Stderr.WriteString("Error: " + err.Error() + "\n")
		os.Exit(1)
	}
}

var initCmd = &cli.Command{
	Name:  "init",
	Usage: "initialize a cli sao client",
	UsageText: "if you want to use sao cli client, you must first init using this command.\n " +
		"create sao chain account locally which will be used as default account in following commands. \n" +
		"under --repo directory, there are client configuration file and keystore.",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:     cliutil.FlagKeyName,
			Usage:    "sao chain account key name",
			Required: true,
			Aliases:  []string{"k"},
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

		accountName, address, mnemonic, err := chain.Create(cctx.Context, cliutil.KeyringHome, saoclient.Cfg.KeyName)
		if err != nil {
			return err
		}
		fmt.Println("account created: ")
		fmt.Println("Account:", accountName)
		fmt.Println("Address:", address)
		fmt.Println("Mnemonic:", mnemonic)

		for {
			coins, err := saoclient.GetBalance(cctx.Context, address)
			askFor := false
			if err != nil {
				fmt.Printf("%v", err)
				askFor = true
			} else {
				if coins.AmountOf("sao").LT(math.NewInt(1000)) {
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

		didManager, address, err := cliutil.GetDidManager(cctx, saoclient.Cfg.KeyName)
		if err != nil {
			return err
		}

		hash, err := saoclient.UpdateDidBinding(cctx.Context, address, didManager.Id, fmt.Sprintf("cosmos:sao:%s", address))
		if err != nil {
			return err
		}

		err = saoclient.SaveConfig(saoclient.Cfg)
		if err != nil {
			return types.Wrapf(types.ErrWriteConfigFailed, "save local config failed: %v", err)
		}

		fmt.Printf("Created DID %s. tx hash %s", didManager.Id, hash)
		fmt.Println()
		fmt.Println("sao client initialized.")
		return nil
	},
}
