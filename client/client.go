package client

import (
	"context"
	"golang.org/x/xerrors"
	"os"
	"path/filepath"
	"sao-node/api"
	"sao-node/chain"
	"sao-node/utils"

	apiclient "sao-node/api/client"

	"github.com/mitchellh/go-homedir"
)

type SaoClientConfig struct {
	GroupId      string
	KeyName      string
	ChainAddress string
	Gateway      string
	Token        string
}

type SaoClient struct {
	api.SaoApi
	chain.ChainSvcApi
	Cfg  *SaoClientConfig
	repo string
}

type SaoClientOptions struct {
	Repo      string
	Gateway   string
	ChainAddr string
	KeyName   string
}

func NewSaoClient(ctx context.Context, opt SaoClientOptions) (*SaoClient, func(), error) {
	cliPath, err := homedir.Expand(opt.Repo)
	if err != nil {
		return nil, nil, err
	}

	// prepare config file
	configPath := filepath.Join(cliPath, "config.toml")
	_, err = os.Stat(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			err = os.MkdirAll(cliPath, 0755) //nolint: gosec
			if err != nil && !os.IsExist(err) {
				return nil, nil, err
			}

			c, err := os.Create(configPath)
			if err != nil {
				return nil, nil, err
			}

			config := defaultSaoClientConfig()
			if opt.Gateway != "" && opt.Gateway != "none" {
				config.Gateway = opt.Gateway
			}
			if opt.ChainAddr != "" && opt.ChainAddr != "none" {
				config.ChainAddress = opt.ChainAddr
			}
			if opt.KeyName != "" {
				config.KeyName = opt.KeyName
			}

			dc, err := utils.NodeBytes(config)
			if err != nil {
				return nil, nil, err
			}
			_, err = c.Write(dc)
			if err != nil {
				return nil, nil, err
			}

			if err := c.Close(); err != nil {
				return nil, nil, err
			}
		}
	}
	c, err := utils.FromFile(configPath, defaultSaoClientConfig())
	if err != nil {
		return nil, nil, err
	}
	cfg, ok := c.(*SaoClientConfig)
	if !ok {
		return nil, nil, xerrors.Errorf("invalid config: %v", c)
	}

	// prepare Gateway api
	var gatewayApi api.SaoApi = nil
	var closer = func() {}
	if opt.Gateway != "none" {
		if opt.Gateway == "" {
			opt.Gateway = cfg.Gateway
		}
		if opt.Gateway == "" {
			return nil, nil, xerrors.Errorf("invalid Gateway")
		}

		if len(cfg.Token) == 0 {
			return nil, nil, xerrors.New("invalid token")
		}

		gatewayApi, closer, err = apiclient.NewGatewayApi(ctx, opt.Gateway, cfg.Token)
		if err != nil {
			return nil, nil, err
		}
	}

	var chainApi chain.ChainSvcApi = nil
	if opt.ChainAddr != "none" {
		// prepare chain svc
		if opt.ChainAddr == "" {
			opt.ChainAddr = cfg.ChainAddress
		}
		chainSvc, err := chain.NewChainSvc(ctx, opt.Repo, "cosmos", opt.ChainAddr, "/websocket")
		if err != nil {
			return nil, nil, xerrors.Errorf("new cosmos chain: %w", err)
		}
		chainApi = chainSvc
	}

	return &SaoClient{
		SaoApi:      gatewayApi,
		ChainSvcApi: chainApi,
		Cfg:         cfg,
		repo:        opt.Repo,
	}, closer, nil
}

func defaultSaoClientConfig() *SaoClientConfig {
	return &SaoClientConfig{
		GroupId:      utils.GenerateGroupId(),
		KeyName:      "",
		ChainAddress: "http://localhost:26657",
		Gateway:      "http://127.0.0.1:5151/rpc/v0",
		Token:        "DEFAULT_TOKEN",
	}
}

func (sc SaoClient) SaveConfig(cfg *SaoClientConfig) error {
	cliPath, err := homedir.Expand(sc.repo)
	if err != nil {
		return err
	}

	configPath := filepath.Join(cliPath, "config.toml")
	c, err := os.OpenFile(configPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0777)
	if err != nil {
		return err
	}

	dc, err := utils.NodeBytes(cfg)
	if err != nil {
		return err
	}
	_, err = c.Write(dc)
	if err != nil {
		return err
	}

	if err := c.Close(); err != nil {
		return err
	}
	return nil
}
