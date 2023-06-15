package main

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

var GatewayApi string
var FlagGateway = &cli.StringFlag{
	Name:        "gateway",
	Usage:       "gateway connection",
	EnvVars:     []string{"SAO_GATEWAY_API"},
	Required:    false,
	Destination: &GatewayApi,
}

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
		Gateway:     GatewayApi,
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
			FlagGateway,
			flagPlatform,
			cliutil.FlagVeryVerbose,
			cliutil.FlagKeyringHome,
		},
		Commands: []*cli.Command{
			initCmd,
			initDidCmd,
			recoverCmd,
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
		"under --repo directory, there is client configuration file,\n" +
		"under --keyring directory, there are keystore files.",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:     cliutil.FlagKeyName,
			Usage:    "sao chain account key name",
			Required: true,
			Aliases:  []string{"k"},
		},
		&cli.StringFlag{
			Name:     "chain-id",
			Required: false,
			Value:    "sao",
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

		hash, err := saoclient.UpdateDidBinding(cctx.Context, address, didManager.Id, fmt.Sprintf("cosmos:%s:%s", cctx.String("chain-id"), address))
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

var initDidCmd = &cli.Command{
	Name:  "initDid",
	Usage: "initialize a Decentralized Identifier (DID) on the sao client",
	UsageText: "To use sao CLI with Decentralized Identifiers (DID), first initialize it using this command.\n " +
		"This creates a sao chain account locally that will be used as default account in subsequent commands. \n" +
		"In the --repo directory, you will find the client configuration file.\n" +
		"In the --keyring directory, you will find the keystore files.",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:     cliutil.FlagKeyName,
			Usage:    "sao chain account key name",
			Required: true,
			Aliases:  []string{"k"},
		},
		&cli.StringFlag{
			Name:     "chain-id",
			Required: false,
			Value:    "sao",
		},
		&cli.StringFlag{
			Name:     "address",
			Required: false,
			Value:    "",
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

		address := cctx.String("address")
		fmt.Println("account created: ")
		fmt.Println("Address:", address)

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
		fmt.Println("DID:", didManager.Id)

		hash, err := saoclient.UpdateDidBinding(cctx.Context, address, didManager.Id, fmt.Sprintf("cosmos:%s:%s", cctx.String("chain-id"), address))
		if err != nil {
			return err
		}

		err = saoclient.SaveConfig(saoclient.Cfg)
		if err != nil {
			return types.Wrapf(types.ErrWriteConfigFailed, "save local config failed: %v", err)
		}

		fmt.Printf("Created DID %s. tx hash %s", didManager.Id, hash)
		return nil
	},
}

var recoverCmd = &cli.Command{
	Name:  "recover",
	Usage: "recover cli sao client with a specific did",
	UsageText: "if you have already init sao cli client, you can do recover client by did.\n " +
		"return error if did is not exists or payment address of did is not found in keyring directory ",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:     cliutil.FlagKeyName,
			Usage:    "sao chain account key name",
			Required: true,
			Aliases:  []string{"k"},
		},
		&cli.StringFlag{
			Name:     "did",
			Usage:    "sao chain key did",
			Required: true,
		},
		&cli.StringFlag{
			Name:     "chain-id",
			Required: false,
			Value:    "sao",
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
		address, err := chain.GetAddress(cctx.Context, cliutil.KeyringHome, saoclient.Cfg.KeyName)
		//accountName, address, mnemonic, err := chain.Create(cctx.Context, cliutil.KeyringHome, saoclient.Cfg.KeyName)
		if err != nil {
			return err
		}

		fmt.Printf("address with key name %s is, %s\n", saoclient.Cfg.KeyName, address)

		did := cctx.String("did")
		payAddr, err := saoclient.QueryPaymentAddress(cctx.Context, did)
		if err != nil {
			return err
		}
		fmt.Printf("payment address of did %s is, %s\n", did, payAddr)

		if address != payAddr {
			return types.ErrInconsistentAddress
		}

		err = saoclient.SaveConfig(saoclient.Cfg)
		if err != nil {
			return types.Wrapf(types.ErrWriteConfigFailed, "save local config failed: %v", err)
		}

		fmt.Println()
		fmt.Println("sao client recovered.")
		return nil
	},
}
