package client

import (
	"context"
	"golang.org/x/xerrors"
	"os"
	"path/filepath"
	"sao-storage-node/api"
	apitypes "sao-storage-node/api/types"
	"sao-storage-node/types"
	"sao-storage-node/utils"

	apiclient "sao-storage-node/api/client"

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
	Cfg        *SaoClientConfig
	gatewayApi api.GatewayApi
	repo       string
}

func NewSaoClient(ctx context.Context, repo string, gatewayAddr string) (*SaoClient, error) {
	cliPath, err := homedir.Expand(repo)
	if err != nil {
		return nil, err
	}

	configPath := filepath.Join(cliPath, "config.toml")
	_, err = os.Stat(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			err = os.MkdirAll(cliPath, 0755) //nolint: gosec
			if err != nil && !os.IsExist(err) {
				return nil, err
			}

			c, err := os.Create(configPath)
			if err != nil {
				return nil, err
			}

			dc, err := utils.NodeBytes(defaultSaoClientConfig())
			if err != nil {
				return nil, err
			}
			_, err = c.Write(dc)
			if err != nil {
				return nil, err
			}

			if err := c.Close(); err != nil {
				return nil, err
			}
		}
	}
	c, err := utils.FromFile(configPath, defaultSaoClientConfig())
	if err != nil {
		return nil, err
	}
	cfg, ok := c.(*SaoClientConfig)
	if !ok {
		return nil, xerrors.Errorf("invalid config: %v", c)
	}

	if gatewayAddr == "none" {
		return &SaoClient{
			Cfg: cfg,
		}, nil
	} else if gatewayAddr == "" {
		gatewayAddr = cfg.Gateway
	}

	if gatewayAddr == "" {
		return nil, xerrors.Errorf("invalid gateway")
	}

	if len(cfg.Token) == 0 {
		return nil, xerrors.New("invalid token")
	}

	gatewayApi, closer, err := apiclient.NewGatewayApi(ctx, gatewayAddr, cfg.Token)
	if err != nil {
		return nil, err
	}
	defer closer()

	return &SaoClient{
		Cfg:        cfg,
		gatewayApi: gatewayApi,
		repo:       repo,
	}, nil
}

func defaultSaoClientConfig() *SaoClientConfig {
	return &SaoClientConfig{
		GroupId:      utils.GenerateGroupId(),
		KeyName:      "",
		ChainAddress: "http://localhost:26657",
		Gateway:      "http://127.0.0.1:8888/rpc/v0",
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

func (sc SaoClient) Test(ctx context.Context) (string, error) {
	resp, err := sc.gatewayApi.Test(ctx, "hello")
	if err != nil {
		return "", err
	}
	return resp, nil
}

func (sc SaoClient) Create(ctx context.Context, req *types.MetadataProposal, orderProposal *types.OrderStoreProposal, orderId uint64, content []byte) (apitypes.CreateResp, error) {
	return sc.gatewayApi.Create(ctx, req, orderProposal, orderId, content)
}

func (sc SaoClient) CreateFile(ctx context.Context, req *types.MetadataProposal, orderProposal *types.OrderStoreProposal, orderId uint64) (apitypes.CreateResp, error) {
	return sc.gatewayApi.CreateFile(ctx, req, orderProposal, orderId)
}

func (sc SaoClient) Load(ctx context.Context, req *types.MetadataProposal) (apitypes.LoadResp, error) {
	return sc.gatewayApi.Load(ctx, req)
}

func (sc SaoClient) Delete(ctx context.Context, req *types.OrderTerminateProposal) (apitypes.DeleteResp, error) {
	return sc.gatewayApi.Delete(ctx, req)
}

func (sc SaoClient) ShowCommits(ctx context.Context, req *types.MetadataProposal) (apitypes.ShowCommitsResp, error) {
	return sc.gatewayApi.ShowCommits(ctx, req)
}

func (sc SaoClient) Update(ctx context.Context, req *types.MetadataProposal, orderProposal *types.OrderStoreProposal, orderId uint64, patch []byte) (apitypes.UpdateResp, error) {
	return sc.gatewayApi.Update(ctx, req, orderProposal, orderId, patch)
}

func (sc SaoClient) GetPeerInfo(ctx context.Context) (apitypes.GetPeerInfoResp, error) {
	return sc.gatewayApi.GetPeerInfo(ctx)
}

func (sc SaoClient) GenerateToken(ctx context.Context, owner string) (apitypes.GenerateTokenResp, error) {
	return sc.gatewayApi.GenerateToken(ctx, owner)
}

func (sc SaoClient) GetHttpUrl(ctx context.Context, dataId string) (apitypes.GetUrlResp, error) {
	return sc.gatewayApi.GetHttpUrl(ctx, dataId)
}

func (sc SaoClient) GetIpfsUrl(ctx context.Context, cid string) (apitypes.GetUrlResp, error) {
	return sc.gatewayApi.GetIpfsUrl(ctx, cid)
}

func (sc SaoClient) NodeAddress(ctx context.Context) (string, error) {
	return sc.gatewayApi.NodeAddress(ctx)
}
