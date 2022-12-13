package account

import (
	"bufio"
	"fmt"
	"os"
	"sao-node/chain"
	cliutil "sao-node/cmd"
	"strings"
	"syscall"

	"github.com/urfave/cli/v2"
	"golang.org/x/term"
)

var AccountCmd = &cli.Command{
	Name:  "account",
	Usage: "account management",
	Subcommands: []*cli.Command{
		listCmd,
		createCmd,
		importCmd,
		exportCmd,
	},
}

var listCmd = &cli.Command{
	Name: "list",
	Action: func(cctx *cli.Context) error {
		ctx := cctx.Context

		err := chain.List(ctx, cliutil.GetChainId())
		if err != nil {
			return err
		}

		return nil
	},
}

var createCmd = &cli.Command{
	Name: "create",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "name",
			Usage: "account name",
		},
	},
	Action: func(cctx *cli.Context) error {
		ctx := cctx.Context

		name := cctx.String("name")
		if !cctx.IsSet("name") {
			reader := bufio.NewReader(os.Stdin)
			fmt.Print("Enter account name:")
			indata, err := reader.ReadBytes('\n')
			if err != nil {
				return err
			}
			name = strings.Replace(string(indata), "\n", "", -1)
		}

		err := chain.Create(ctx, cliutil.GetChainId(), name)
		if err != nil {
			return err
		}

		return nil
	},
}

var exportCmd = &cli.Command{
	Name: "export",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "name",
			Usage: "account name",
		},
	},
	Action: func(cctx *cli.Context) error {
		ctx := cctx.Context

		name := cctx.String("name")
		if !cctx.IsSet("name") {
			reader := bufio.NewReader(os.Stdin)
			fmt.Print("Enter account name:")
			indata, err := reader.ReadBytes('\n')
			if err != nil {
				return err
			}
			name = strings.Replace(string(indata), "\n", "", -1)
		}

		fmt.Print("Enter passphrase:")
		passphrase, err := term.ReadPassword(syscall.Stdin)
		if err != nil {
			return err
		}

		err = chain.Export(ctx, cliutil.GetChainId(), name, string(passphrase))
		if err != nil {
			return err
		}

		return nil
	},
}

var importCmd = &cli.Command{
	Name: "import",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "name",
			Usage: "account name",
		},
	},
	Action: func(cctx *cli.Context) error {
		ctx := cctx.Context

		name := cctx.String("name")
		if !cctx.IsSet("name") {
			reader := bufio.NewReader(os.Stdin)
			fmt.Print("Enter account name:")
			indata, err := reader.ReadBytes('\n')
			if err != nil {
				return err
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
				return err
			}

			secret = secret + line

			if strings.Contains(line, "-----END TENDERMINT PRIVATE KEY-----") {
				break
			}
		}

		fmt.Print("Enter passphrase:")
		passphrase, err := term.ReadPassword(syscall.Stdin)
		if err != nil {
			return err
		}

		err = chain.Import(ctx, cliutil.GetChainId(), name, secret, string(passphrase))
		if err != nil {
			return err
		}

		return nil
	},
}
