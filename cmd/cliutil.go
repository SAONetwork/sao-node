package cliutil

import (
	"fmt"
	"golang.org/x/term"
	"sao-storage-node/chain"
	saoclient "sao-storage-node/client"
	"syscall"

	saodid "github.com/SaoNetwork/sao-did"
	saokey "github.com/SaoNetwork/sao-did/key"
	"github.com/urfave/cli/v2"
)

const (
	SECP256K1 = "secp256k1"
)

const (
	devNetChainId  = "sao"
	testNetChainId = "sao-testnet-fcf77b"
	mainNetChainId = "sao"
)

var ChainAddress string
var FlagChainAddress = &cli.StringFlag{
	Name:        "chain-address",
	EnvVars:     []string{"SAO_CHAIN_API"},
	Value:       "http://localhost:26657",
	Destination: &ChainAddress,
}

var NetType string
var FlagNetType = &cli.StringFlag{
	Name:        "net",
	Usage:       "sao network type: [devnet/testnet/mainnet]",
	Value:       "testnet",
	Destination: &NetType,
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

func AskForPassphrase() (string, error) {
	fmt.Print("Enter passphrase:")
	passphrase, err := term.ReadPassword(syscall.Stdin)
	if err != nil {
		return "", err
	}
	return string(passphrase), nil
}

func GetDidManager(cctx *cli.Context, cfg *saoclient.SaoClientConfig) (*saodid.DidManager, string, error) {
	var keyName string
	if !cctx.IsSet("keyName") {
		keyName = cfg.KeyName
	} else {
		keyName = cctx.String("keyName")
	}

	address, err := chain.GetAddress(cctx.Context, "sao", keyName)
	if err != nil {
		return nil, "", err
	}

	payload := fmt.Sprintf("cosmos %s allows to generate did", address)
	secret, err := chain.SignByAccount(cctx.Context, "sao", keyName, []byte(payload))
	if err != nil {
		return nil, "", err
	}

	provider, err := saokey.NewSecp256k1Provider(secret)
	if err != nil {
		return nil, "", err
	}
	resolver := saokey.NewKeyResolver()

	didManager := saodid.NewDidManager(provider, resolver)
	_, err = didManager.Authenticate([]string{}, "")
	if err != nil {
		return nil, "", err
	}

	cfg.KeyName = keyName
	return &didManager, address, nil
}

func GetChainId() string {
	switch NetType {
	case "devnet":
		return devNetChainId
	case "testnet":
		return testNetChainId
	case "mainnet":
		return mainNetChainId
	}
	return devNetChainId
}
