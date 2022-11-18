package cliutil

import (
	"encoding/hex"

	saodid "github.com/SaoNetwork/sao-did"
	saokey "github.com/SaoNetwork/sao-did/key"
	saodidtypes "github.com/SaoNetwork/sao-did/types"
	"github.com/urfave/cli/v2"
	"golang.org/x/xerrors"
)

const (
	SECP256K1 = "secp256k1"
)

const (
	devNetChainId  = "sao"
	testNetChainId = "sao-testnet-fcf77b"
	mainNetChainId = "sao"
)

var NetType = &cli.StringFlag{
	Name:  "net",
	Usage: "sao network type: [devnet/testnet/mainnet]",
	Value: "testnet",
}

// IsVeryVerbose is a global var signalling if the CLI is running in very
// verbose mode or not (default: false).
var IsVeryVerbose bool

// FlagVeryVerbose enables very verbose mode, which is useful when debugging
// the CLI itself. It should be included as a flag on the top-level command
// (e.g. lotus -vv, lotus-miner -vv).
var FlagVeryVerbose = &cli.BoolFlag{
	Name:        "vv",
	Usage:       "enables very verbose mode, useful for debugging the CLI",
	Destination: &IsVeryVerbose,
}

func GetDidManager(cctx *cli.Context, defaultSeed string, defaultAlg string) (*saodid.DidManager, error) {
	var secret []byte
	var err error
	if cctx.IsSet("secret") {
		seed := cctx.String("secret")
		secret, err = hex.DecodeString(seed)
		if err != nil {
			return nil, err
		}
	} else {
		secret, err = hex.DecodeString(defaultSeed)
		if err != nil {
			return nil, err
		}
	}

	alg := defaultAlg
	if cctx.IsSet("alg") {
		alg := cctx.String("alg")
		if alg != SECP256K1 {
			return nil, xerrors.Errorf("unsupported alg %s", alg)
		}
	}

	var provider saodidtypes.DidProvider
	if alg == SECP256K1 {
		provider, err = saokey.NewSecp256k1Provider(secret)
		if err != nil {
			return nil, err
		}
	}

	didManager := saodid.NewDidManager(provider, saokey.NewKeyResolver())
	_, err = didManager.Authenticate([]string{}, "")
	if err != nil {
		return nil, err
	}
	return &didManager, nil
}

func GetChainId(cctx *cli.Context) string {
	switch cctx.String("net") {
	case "devnet":
		return devNetChainId
	case "testnet":
		return testNetChainId
	case "mainnet":
		return mainNetChainId
	}
	return devNetChainId
}
