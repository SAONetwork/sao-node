package main

import (
	"encoding/hex"
	"fmt"
	"golang.org/x/term"
	"sao-storage-node/chain"
	saoclient "sao-storage-node/client"
	cliutil "sao-storage-node/cmd"
	"syscall"

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
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:     "keyname",
			Required: true,
		},
	},
	Action: func(cctx *cli.Context) error {
		keyName := cctx.String("keyname")
		fmt.Print("Enter passphrase:")
		passphrase, err := term.ReadPassword(syscall.Stdin)
		if err != nil {
			return err
		}

		saoclient := saoclient.NewSaoClient(cctx.Context, "none")

		chainAddress := cliutil.ChainAddress
		if chainAddress == "" {
			chainAddress = saoclient.Cfg.ChainAddress
		}

		alg := saoclient.Cfg.Alg
		if alg == cliutil.SECP256K1 {
			address, secret, err := chain.GetAccountSecret(cctx.Context, "sao", keyName, string(passphrase))
			if err != nil {
				return err
			}
			secretBytes, err := hex.DecodeString(secret)
			//secret, err := hex.DecodeString(saoclient.Cfg.Seed)
			if err != nil {
				return err
			}
			provider, err := saokey.NewSecp256k1Provider(secretBytes)
			if err != nil {
				return err
			}
			resolver := saokey.NewKeyResolver()
			didManager := did.NewDidManager(provider, resolver)
			_, err = didManager.Authenticate([]string{}, "")
			if err != nil {
				return err
			}

			chainSvc, err := chain.NewChainSvc(cctx.Context, "cosmos", chainAddress, "/websocket")
			hash, err := chainSvc.UpdateDidBinding(cctx.Context, address, didManager.Id, fmt.Sprintf("cosmos:sao:%s", address))
			if err != nil {
				return err
			}

			fmt.Printf("Created DID %s with key %s. tx hash %s", didManager.Id, keyName, hash)
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
		client := saoclient.NewSaoClient(cctx.Context, "none")
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
