package account

import (
	"bufio"
	"fmt"
	"os"
	"sao-node/chain"
	cliutil "sao-node/cmd"
	"sao-node/types"
	"strings"
	"syscall"

	"github.com/labstack/gommon/log"
	"github.com/mitchellh/go-homedir"
	"github.com/urfave/cli/v2"
	"golang.org/x/term"
)

var AccountCmd = &cli.Command{
	Name:  "account",
	Usage: "account management",
	Subcommands: []*cli.Command{
		listCmd,
		createCmd,
		sendCmd,
		importCmd,
		exportCmd,
	},
}

var listCmd = &cli.Command{
	Name:  "list",
	Usage: "list all sao chain account in local keystore",
	Action: func(cctx *cli.Context) error {
		ctx := cctx.Context

		repoPath := cctx.String("repo")
		if repoPath == "" {
			var err error
			if cctx.App.Name == "saoclient" {
				repoPath, err = homedir.Expand("~/.sao-cli")
			} else if cctx.App.Name == "saonode" {
				repoPath, err = homedir.Expand("~/.sao-node")
			} else {
				return types.Wrapf(types.ErrInvalidBinaryName, ", Name=%s", cctx.App.Name)
			}
			if err != nil {
				return types.Wrapf(types.ErrInvalidRepoPath, ", path=%s, %w", err)
			}
		}
		chainAddress, err := cliutil.GetChainAddress(cctx, repoPath)
		if err != nil {
			log.Warn(err)
		}

		chain, err := chain.NewChainSvc(ctx, repoPath, "cosmos", chainAddress, "/websocket", cliutil.KeyringHome)
		if err != nil {
			return types.Wrap(types.ErrCreateChainServiceFailed, err)
		}
		err = chain.List(ctx, cliutil.KeyringHome)
		if err != nil {
			return err
		}

		return nil
	},
}

var createCmd = &cli.Command{
	Name:  "create",
	Usage: "create a new local account with the given name",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:     cliutil.FlagKeyName,
			Usage:    "account name",
			Required: true,
		},
	},
	Action: func(cctx *cli.Context) error {
		ctx := cctx.Context

		name := cctx.String(cliutil.FlagKeyName)
		if !cctx.IsSet(cliutil.FlagKeyName) {
			reader := bufio.NewReader(os.Stdin)
			fmt.Print("Enter account name:")
			indata, err := reader.ReadBytes('\n')
			if err != nil {
				return types.Wrap(types.ErrAccountNotFound, err)
			}
			name = strings.Replace(string(indata), "\n", "", -1)
		}

		repoPath := cctx.String("repo")
		if repoPath == "" {
			if cctx.App.Name == "saoclient" {
				repoPath = "~/.sao-cli"
			} else if cctx.App.Name == "saonode" {
				repoPath = "~/.sao-node"
			} else {
				return types.Wrapf(types.ErrInvalidBinaryName, ", Name=%s", cctx.App.Name)
			}
		}

		accountName, address, mnemonic, err := chain.Create(ctx, cliutil.KeyringHome, name)
		if err != nil {
			return err
		}
		fmt.Println("Account: ", accountName)
		fmt.Println("Address: ", address)
		fmt.Println("Mnemonic: ", mnemonic)
		fmt.Println()

		return nil
	},
}

var exportCmd = &cli.Command{
	Name:  "export",
	Usage: "Export the given local account's encrypted private key",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:     cliutil.FlagKeyName,
			Usage:    "account name to export",
			Required: true,
		},
	},
	Action: func(cctx *cli.Context) error {
		ctx := cctx.Context

		name := cctx.String(cliutil.FlagKeyName)
		if !cctx.IsSet(cliutil.FlagKeyName) {
			reader := bufio.NewReader(os.Stdin)
			fmt.Print("Enter account name:")
			indata, err := reader.ReadBytes('\n')
			if err != nil {
				return types.Wrap(types.ErrAccountNotFound, err)
			}
			name = strings.Replace(string(indata), "\n", "", -1)
		}

		fmt.Print("Enter passphrase:")
		passphrase, err := term.ReadPassword(syscall.Stdin)
		if err != nil {
			return err
		}

		repoPath := cctx.String("repo")
		if repoPath == "" {
			if cctx.App.Name == "saoclient" {
				repoPath = "~/.sao-cli"
			} else if cctx.App.Name == "saonode" {
				repoPath = "~/.sao-node"
			} else {
				return types.Wrapf(types.ErrInvalidBinaryName, ", Name=%s", cctx.App.Name)
			}
		}

		err = chain.Export(ctx, cliutil.KeyringHome, name, string(passphrase))
		if err != nil {
			return err
		}

		return nil
	},
}

var sendCmd = &cli.Command{
	Name:  "send",
	Usage: "send SAO tokens from one account to another",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:     "from",
			Usage:    "the original account to spend tokens",
			Required: true,
		},
		&cli.StringFlag{
			Name:     "to",
			Usage:    "the target account to received tokens",
			Required: true,
		},
		&cli.Int64Flag{
			Name:     "amount",
			Usage:    "the token amount to send",
			Required: true,
		},
	},
	Action: func(cctx *cli.Context) error {
		ctx := cctx.Context

		repoPath := cctx.String("repo")
		if repoPath == "" {
			if cctx.App.Name == "saoclient" {
				repoPath = "~/.sao-cli"
			} else if cctx.App.Name == "saonode" {
				repoPath = "~/.sao-node"
			} else {
				return types.Wrapf(types.ErrInvalidBinaryName, ", Name=%s", cctx.App.Name)
			}
		}

		chainAddress, err := cliutil.GetChainAddress(cctx, repoPath)
		if err != nil {
			log.Warn(err)
		}

		chain, err := chain.NewChainSvc(ctx, repoPath, "cosmos", chainAddress, "/websocket", cliutil.KeyringHome)
		if err != nil {
			return types.Wrap(types.ErrCreateChainServiceFailed, err)
		}
		from := cctx.String("from")
		to := cctx.String("to")
		amount := cctx.Int64("amount")
		txHash, err := chain.Send(ctx, from, to, amount)
		if err != nil {
			return err
		}
		fmt.Printf("%d stakes has been transferred from %s to %s, txHash=%s\n", amount, from, to, txHash)

		return nil
	},
}

var importCmd = &cli.Command{
	Name: "import",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:     cliutil.FlagKeyName,
			Usage:    "account name to import",
			Required: true,
		},
	},
	Action: func(cctx *cli.Context) error {
		ctx := cctx.Context

		name := cctx.String(cliutil.FlagKeyName)
		if !cctx.IsSet(cliutil.FlagKeyName) {
			reader := bufio.NewReader(os.Stdin)
			fmt.Print("Enter account name:")
			indata, err := reader.ReadBytes('\n')
			if err != nil {
				return types.Wrap(types.ErrAccountNotFound, err)
			}
			name = strings.Replace(string(indata), "\n", "", -1)
		}

		fmt.Println("Enter secret:")
		var secret string
		reader := bufio.NewReader(os.Stdin)
		for {
			// read line from stdin using newline as separator
			line, err := reader.ReadString('\n')
			if err != nil {
				return types.Wrap(types.ErrInvalidSecrect, err)
			}

			secret = secret + line

			if strings.Contains(line, "-----END TENDERMINT PRIVATE KEY-----") {
				break
			}
		}

		fmt.Print("Enter passphrase:")
		passphrase, err := term.ReadPassword(syscall.Stdin)
		if err != nil {
			return types.Wrap(types.ErrInvalidPassphrase, err)
		}

		repoPath := cctx.String("repo")
		if repoPath == "" {
			if cctx.App.Name == "saoclient" {
				repoPath = "~/.sao-cli"
			} else if cctx.App.Name == "saonode" {
				repoPath = "~/.sao-node"
			} else {
				return types.Wrapf(types.ErrInvalidBinaryName, ", Name=%s", cctx.App.Name)
			}
		}
		err = chain.Import(ctx, cliutil.KeyringHome, name, secret, string(passphrase))
		if err != nil {
			return err
		}

		return nil
	},
}
