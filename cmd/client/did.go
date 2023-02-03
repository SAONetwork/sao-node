package main

import (
	"fmt"
	"sao-node/chain"
	saoclient "sao-node/client"
	cliutil "sao-node/cmd"

	"github.com/tendermint/tendermint/libs/json"
	"github.com/urfave/cli/v2"
)

var didCmd = &cli.Command{
	Name:  "did",
	Usage: "did management",
	Subcommands: []*cli.Command{
		didCreateCmd,
		didShowInfoCmd,
		didSignCmd,
	},
}

var didCreateCmd = &cli.Command{
	Name:  "create",
	Usage: "create a new did based on the given sao account.",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:     cliutil.FlagKeyName,
			Usage:    "sao chain key name which did will be generated on",
			Required: true,
		},
		&cli.BoolFlag{
			Name:     "override",
			Usage:    "override default client configuration's key account.",
			Required: false,
			Value:    false,
		},
	},
	Action: func(cctx *cli.Context) error {
		opt := saoclient.SaoClientOptions{
			Repo:        cctx.String(FlagClientRepo),
			Gateway:     "none",
			ChainAddr:   cliutil.ChainAddress,
			KeyringHome: cliutil.KeyringHome,
		}
		saoclient, closer, err := saoclient.NewSaoClient(cctx.Context, opt)
		if err != nil {
			return err
		}
		defer closer()

		didManager, address, err := cliutil.GetDidManager(cctx, saoclient.Cfg)
		if err != nil {
			return err
		}

		hash, err := saoclient.UpdateDidBinding(cctx.Context, address, didManager.Id, fmt.Sprintf("cosmos:sao:%s", address))
		if err != nil {
			return err
		}

		if cctx.Bool("override") {
			err = saoclient.SaveConfig(saoclient.Cfg)
			if err != nil {
				return fmt.Errorf("save local config failed: %v", err)
			}
		}

		fmt.Printf("Created DID %s. tx hash %s", didManager.Id, hash)
		fmt.Println()
		return nil
	},
}

var didShowInfoCmd = &cli.Command{
	Name:  "info",
	Usage: "show did information",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:     "did-url",
			Usage:    "did URL",
			Required: true,
		},
	},
	Action: func(cctx *cli.Context) error {
		ctx := cctx.Context

		repoPath := cctx.String("repo")
		chainAddress := cliutil.ChainAddress
		if chainAddress == "" {
			return fmt.Errorf("no chain address specified")
		}

		chain, err := chain.NewChainSvc(ctx, repoPath, "cosmos", chainAddress, "/websocket", cliutil.KeyringHome)
		if err != nil {
			return fmt.Errorf("new cosmos chain: %w", err)
		}
		chain.ShowDidInfo(ctx, cctx.String("did-url"))

		return nil
	},
}

var didSignCmd = &cli.Command{
	Name:  "sign",
	Usage: "using the given did to sign a payload",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:     cliutil.FlagKeyName,
			Usage:    "sao chain key name which did will be generated on",
			Required: true,
		},
	},
	Action: func(cctx *cli.Context) error {
		opt := saoclient.SaoClientOptions{
			Repo:        cctx.String(FlagClientRepo),
			Gateway:     "none",
			ChainAddr:   "none",
			KeyringHome: cliutil.KeyringHome,
		}
		saoclient, closer, err := saoclient.NewSaoClient(cctx.Context, opt)
		if err != nil {
			return err
		}
		defer closer()

		didManager, _, err := cliutil.GetDidManager(cctx, saoclient.Cfg)
		if err != nil {
			return err
		}

		jws, err := didManager.CreateJWS([]byte(cctx.Args().First()))
		if err != nil {
			return err
		}

		j, err := json.MarshalIndent(jws, "", "    ")
		if err != nil {
			return err
		}
		fmt.Println(string(j))
		return nil
	},
}
