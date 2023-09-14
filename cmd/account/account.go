package account

import (
	"bufio"
	"fmt"
	"github.com/cosmos/go-bip39"
	"os"
	"strings"
	"syscall"

	"github.com/SaoNetwork/sao-node/chain"
	cliutil "github.com/SaoNetwork/sao-node/cmd"
	"github.com/SaoNetwork/sao-node/types"

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

		var err error
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
		repoPath, err = homedir.Expand(repoPath)
		if err != nil {
			return types.Wrapf(types.ErrInvalidRepoPath, ", path=%s, %v", err)
		}
		chainAddress, err := cliutil.GetChainAddress(cctx, repoPath, cctx.App.Name)
		if err != nil {
			log.Warn(err)
		}

		chain, err := chain.NewChainSvc(ctx, chainAddress, "/websocket", cliutil.KeyringHome)
		if err != nil {
			return err
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

		chainAddress, err := cliutil.GetChainAddress(cctx, repoPath, cctx.App.Name)
		if err != nil {
			log.Warn(err)
		}

		chain, err := chain.NewChainSvc(ctx, chainAddress, "/websocket", cliutil.KeyringHome)
		if err != nil {
			return err
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

		fmt.Println("Enter secret (Mnemonic or Tendermint Private Key):")
		reader := bufio.NewReader(os.Stdin)
		var secret string
		var mnemonic string

		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				return types.Wrap(types.ErrInvalidSecrect, err)
			}

			secret = secret + line

			if strings.HasPrefix(secret, "-----BEGIN TENDERMINT PRIVATE KEY-----") && strings.Contains(line, "-----END TENDERMINT PRIVATE KEY-----") {
				break
			}

			if bip39.IsMnemonicValid(strings.TrimSpace(secret)) {
				mnemonic = strings.TrimSpace(secret)
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

		if mnemonic != "" {
			addr, err := chain.GenerateAccount(ctx, cliutil.KeyringHome, name, mnemonic)
			if err != nil {
				return err
			}
			fmt.Printf("Account %s has been imported, address=%s\n", name, addr)
			// Additional code to handle successful mnemonic import
			// ...
		} else {
			err = chain.Import(ctx, cliutil.KeyringHome, name, secret, string(passphrase))
			if err != nil {
				return err
			}
		}

		return nil
	},
}
