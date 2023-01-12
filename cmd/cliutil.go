package cliutil

import (
	"fmt"
	"os"
	"path/filepath"
	"sao-node/chain"
	saoclient "sao-node/client"
	gen "sao-node/gen/clidoc"
	"sao-node/node/config"
	"sao-node/node/repo"
	"sao-node/utils"
	"strings"
	"syscall"

	"golang.org/x/term"
	"golang.org/x/xerrors"

	saodid "github.com/SaoNetwork/sao-did"
	saokey "github.com/SaoNetwork/sao-did/key"
	"github.com/opentracing/opentracing-go/log"
	"github.com/urfave/cli/v2"
)

const FlagKeyName = "key-name"

const (
	FlagStorageRepo        = "repo"
	FlagStorageDefaultRepo = "~/.sao-node"
)

var ChainAddress string
var FlagChainAddress = &cli.StringFlag{
	Name:        "chain-address",
	Usage:       "sao chain api",
	EnvVars:     []string{"SAO_CHAIN_API"},
	Value:       "http://192.168.50.66:26657",
	Destination: &ChainAddress,
}

// IsVeryVerbose is a global var signalling if the CLI is running in very
// verbose mode or not (default: false).
var IsVeryVerbose bool

// FlagVeryVerbose enables very verbose mode, which is useful when debugging
// the CLI itself. It should be included as a flag on the top-level command
// (e.g. saonode -vv).
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

func GetDidManager(cctx *cli.Context, keyName string) (*saodid.DidManager, string, error) {
	if cctx.IsSet(FlagKeyName) {
		keyName = cctx.String(FlagKeyName)
	}

	repo := cctx.String("repo")

	address, err := chain.GetAddress(cctx.Context, repo, keyName)
	if err != nil {
		return nil, "", err
	}

	payload := fmt.Sprintf("cosmos %s allows to generate did", address)
	secret, err := chain.SignByAccount(cctx.Context, repo, keyName, []byte(payload))
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

	return &didManager, address, nil
}

// TODO: move to makefile
var GenerateDocCmd = &cli.Command{
	Name:   "clidoc",
	Hidden: true,
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:     "output",
			Usage:    "file path to export to",
			Required: false,
		},
		&cli.StringFlag{
			Name:     "doctype",
			Usage:    "current supported type: markdown / man",
			Required: false,
			Value:    "markdown",
		},
	},
	Action: func(cctx *cli.Context) error {
		var output string
		var err error
		if cctx.String("doctype") == "markdown" {
			output, err = gen.ToMarkdown(cctx.App)
		} else {
			output, err = cctx.App.ToMan()
		}
		if err != nil {
			return err
		}
		outputFile := cctx.String("output")
		if outputFile == "" {
			outputFile = fmt.Sprintf("./docs/%s.md", cctx.App.Name)
		}
		err = os.WriteFile(outputFile, []byte(output), 0644)
		if err != nil {
			return err
		}
		fmt.Printf("markdown clidoc is exported to %s", outputFile)
		fmt.Println()
		return nil
	},
}

func GetChainAddress(cctx *cli.Context) (string, error) {
	chainAddress := ChainAddress
	repoPath := cctx.String(FlagStorageRepo)
	configPath := filepath.Join(repoPath, "config.toml")
	if strings.Contains(repoPath, "-node") {
		r, err := repo.PrepareRepo(repoPath)
		if err != nil {
			log.Error(err)
			return chainAddress, nil
		}

		c, err := r.Config()
		if err != nil {
			return chainAddress, xerrors.Errorf("invalid config for repo, got: %T", c)
		}

		cfg, ok := c.(*config.Node)
		if !ok {
			return chainAddress, xerrors.Errorf("invalid config for repo, got: %T", c)
		}

		chainAddress = cfg.Chain.Remote
	} else if strings.Contains(repoPath, "-cli") {
		c, err := utils.FromFile(configPath, saoclient.DefaultSaoClientConfig())
		if err != nil {
			return chainAddress, err
		}
		cfg, ok := c.(*saoclient.SaoClientConfig)
		if !ok {
			return chainAddress, xerrors.Errorf("invalid config: %v", c)
		}
		chainAddress = cfg.ChainAddress
	}

	if chainAddress == "" {
		return chainAddress, xerrors.Errorf("no chain address specified")
	}

	return chainAddress, nil
}
