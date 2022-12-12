package main

import (
	"fmt"
	"sao-storage-node/chain"
	saoclient "sao-storage-node/client"
	cliutil "sao-storage-node/cmd"

	"github.com/tendermint/tendermint/libs/json"
	"github.com/urfave/cli/v2"
)

var didCmd = &cli.Command{
	Name:  "did",
	Usage: "did tools",
	Subcommands: []*cli.Command{
		didCreateCmd,
		didSignCmd,
	},
}

var didCreateCmd = &cli.Command{
	Name: "create",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:     "key-name",
			Required: true,
			Usage:    "cosmos key name which did will be generated on",
		},
	},
	Action: func(cctx *cli.Context) error {
		saoclient, err := saoclient.NewSaoClient(cctx.Context, cctx.String(FlagClientRepo), "none")
		if err != nil {
			return err
		}

		chainAddress := cliutil.ChainAddress
		if chainAddress == "" {
			chainAddress = saoclient.Cfg.ChainAddress
		}

		didManager, address, err := cliutil.GetDidManager(cctx, saoclient.Cfg)
		if err != nil {
			return err
		}

		chainSvc, err := chain.NewChainSvc(cctx.Context, "cosmos", chainAddress, "/websocket")
		if err != nil {
			return err
		}
		hash, err := chainSvc.UpdateDidBinding(cctx.Context, address, didManager.Id, fmt.Sprintf("cosmos:sao:%s", address))
		if err != nil {
			return err
		}

		err = saoclient.SaveConfig(saoclient.Cfg)
		if err != nil {
			return fmt.Errorf("save local config failed: %v", err)
		}

		fmt.Printf("Created DID %s. tx hash %s", didManager.Id, hash)
		fmt.Println()
		return nil
	},
}

var didSignCmd = &cli.Command{
	Name: "sign",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:     "key-name",
			Required: true,
		},
	},
	Action: func(cctx *cli.Context) error {
		saoclient, err := saoclient.NewSaoClient(cctx.Context, cctx.String(FlagClientRepo), "none")
		if err != nil {
			return err
		}

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
