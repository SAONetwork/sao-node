package client

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sao-node/api"
	"sao-node/chain"
	"sao-node/types"
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
	Repo        string
	Gateway     string
	ChainAddr   string
	KeyName     string
	KeyringHome string
}

func NewSaoClient(ctx context.Context, opt SaoClientOptions) (*SaoClient, func(), error) {
	cliPath, err := homedir.Expand(opt.Repo)
	if err != nil {
		return nil, nil, types.Wrapf(types.ErrInvalidRepoPath, ", path=%s, %v", err)
	}

	// prepare config file
	configPath := filepath.Join(cliPath, "config.toml")
	_, err = os.Stat(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			err = os.MkdirAll(cliPath, 0755) //nolint: gosec
			if err != nil && !os.IsExist(err) {
				return nil, nil, types.Wrap(types.ErrCreateDirFailed, err)
			}

			c, err := os.Create(configPath)
			if err != nil {
				return nil, nil, types.Wrap(types.ErrCreateFileFailed, err)
			}

			config := DefaultSaoClientConfig()
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
				return nil, nil, types.Wrap(types.ErrEncodeConfigFailed, err)
			}
			_, err = c.Write(dc)
			if err != nil {
				return nil, nil, types.Wrap(types.ErrWriteConfigFailed, err)
			}

			if err := c.Close(); err != nil {
				return nil, nil, types.Wrap(types.ErrCloseFileFailed, err)
			}
		}
	}

	c, err := utils.FromFile(configPath, DefaultSaoClientConfig())
	if err != nil {
		return nil, nil, types.Wrap(types.ErrDecodeConfigFailed, err)
	}
	cfg, ok := c.(*SaoClientConfig)
	if !ok {
		return nil, nil, types.Wrapf(types.ErrReadConfigFailed, "invalid config: %v", c)
	}

	// prepare Gateway api
	var gatewayApi api.SaoApi = nil
	var closer = func() {}
	fmt.Println("opt.gateway", opt.Gateway)
	if opt.Gateway != "none" {
		if opt.Gateway == "" {
			opt.Gateway = cfg.Gateway
		}
		if opt.Gateway == "" {
			return nil, nil, types.Wrap(types.ErrInvalidGateway, err)
		}

		if len(cfg.Token) == 0 {
			return nil, nil, types.Wrapf(types.ErrInvalidToken, "Please fill token in configuration file.")
		}

		gatewayApi, closer, err = apiclient.NewNodeApi(ctx, opt.Gateway, cfg.Token)
		if err != nil {
			return nil, nil, types.Wrap(types.ErrCreateApiServiceFailed, err)
		}
	}

	var chainApi chain.ChainSvcApi = nil
	if opt.ChainAddr != "none" {
		// prepare chain svc
		if opt.ChainAddr == "" {
			opt.ChainAddr = cfg.ChainAddress
		}
		chainSvc, err := chain.NewChainSvc(ctx, opt.ChainAddr, "/websocket", opt.KeyringHome)
		if err != nil {
			return nil, nil, err
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

func DefaultSaoClientConfig() *SaoClientConfig {
	return &SaoClientConfig{
		GroupId:      utils.GenerateGroupId(),
		KeyName:      "",
		ChainAddress: "http://127.0.0.1:26657",
		Gateway:      "http://127.0.0.1:5151/rpc/v0",
		Token:        "",
	}
}

func (sc SaoClient) SaveConfig(cfg *SaoClientConfig) error {
	cliPath, err := homedir.Expand(sc.repo)
	if err != nil {
		return types.Wrapf(types.ErrInvalidRepoPath, ", path=%s, %v", cliPath, err)
	}

	configPath := filepath.Join(cliPath, "config.toml")
	c, err := os.OpenFile(configPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0777)
	if err != nil {
		return types.Wrap(types.ErrOpenFileFailed, err)
	}

	dc, err := utils.NodeBytes(cfg)
	if err != nil {
		return types.Wrap(types.ErrEncodeConfigFailed, err)
	}
	_, err = c.Write(dc)
	if err != nil {
		return types.Wrap(types.ErrWriteConfigFailed, err)
	}

	if err := c.Close(); err != nil {
		return types.Wrap(types.ErrCloseFileFailed, err)
	}
	return nil
}
