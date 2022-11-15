package main

import (
	"encoding/hex"
	"fmt"
	saoclient "sao-storage-node/client"
	cliutil "sao-storage-node/cmd"

	did "github.com/SaoNetwork/sao-did"
	"github.com/tendermint/tendermint/libs/json"
	"github.com/urfave/cli/v2"
	"golang.org/x/xerrors"

	saokey "github.com/SaoNetwork/sao-did/key"
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
	Action: func(cctx *cli.Context) error {
		saoclient := saoclient.NewSaoClient(cctx.Context, "")

		alg := saoclient.Cfg.Alg
		if alg == cliutil.SECP256K1 {
			secret, err := hex.DecodeString(saoclient.Cfg.Seed)
			if err != nil {
				return err
			}
			provider, err := saokey.NewSecp256k1Provider(secret)
			if err != nil {
				return err
			}
			resolver := saokey.NewKeyResolver()
			didManager := did.NewDidManager(provider, resolver)
			_, err = didManager.Authenticate([]string{}, "")
			if err != nil {
				return err
			}
			fmt.Printf("Created DID %s with seed %s", didManager.Id, saoclient.Cfg.Seed)
			fmt.Println()
		} else {
			return xerrors.Errorf("Unsupported alg: %s", alg)
		}
		return nil
	},
}

var didSignCmd = &cli.Command{
	Name: "sign",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:     "key",
			Required: true,
		},
		&cli.StringFlag{
			Name:     "alg",
			Required: false,
			Usage:    "secp256k1",
			Value:    "secp256k1",
		},
	},
	Action: func(cctx *cli.Context) error {
		client := saoclient.NewSaoClient(cctx.Context, "")
		didManager, err := cliutil.GetDidManager(cctx, client.Cfg.Seed, client.Cfg.Alg)
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
