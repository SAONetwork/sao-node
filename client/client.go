package client

import (
	"context"
	"encoding/hex"
	"github.com/thanhpk/randstr"
	"os"
	"path/filepath"
	"sao-storage-node/api"
	apitypes "sao-storage-node/api/types"
	"sao-storage-node/types"
	"sao-storage-node/utils"

	"github.com/mitchellh/go-homedir"
)

type SaoClientConfig struct {
	GroupId string
	Seed    string
	Alg     string
}

type SaoClient struct {
	Cfg        *SaoClientConfig
	gatewayApi api.GatewayApi
}

func NewSaoClient(api api.GatewayApi) *SaoClient {
	cliPath, err := homedir.Expand(SAO_CLI_PATH)
	if err != nil {
		log.Error(err.Error())
		return nil
	}

	configPath := filepath.Join(cliPath, "config.toml")
	_, err = os.Stat(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			err = os.MkdirAll(cliPath, 0755) //nolint: gosec
			if err != nil && !os.IsExist(err) {
				log.Error(err.Error())
				return nil
			}

			c, err := os.Create(configPath)
			if err != nil {
				log.Error(err.Error())
				return nil
			}

			dc, err := utils.NodeBytes(defaultSaoClientConfig())
			if err != nil {
				log.Error(err.Error())
				return nil
			}
			_, err = c.Write(dc)
			if err != nil {
				log.Error(err.Error())
				return nil
			}

			if err := c.Close(); err != nil {
				log.Error(err.Error())
				return nil
			}
		}
	}
	c, err := utils.FromFile(configPath, defaultSaoClientConfig())
	if err != nil {
		log.Error(err.Error())
		return nil
	}
	cfg, ok := c.(*SaoClientConfig)
	if !ok {
		log.Error("invalid config: ", c)
		return nil
	}

	return &SaoClient{
		Cfg:        cfg,
		gatewayApi: api,
	}
}

func defaultSaoClientConfig() *SaoClientConfig {
	return &SaoClientConfig{
		GroupId: utils.GenerateGroupId(),
		Alg:     "secp256k1",
		Seed:    hex.EncodeToString(randstr.Bytes(32)),
	}
}

func (sc SaoClient) Test(ctx context.Context) (string, error) {
	resp, err := sc.gatewayApi.Test(ctx, "hello")
	if err != nil {
		return "", err
	}
	return resp, nil
}

func (sc SaoClient) Create(ctx context.Context, orderProposal types.ClientOrderProposal, orderMeta types.OrderMeta, content []byte) (apitypes.CreateResp, error) {
	return sc.gatewayApi.Create(ctx, orderProposal, orderMeta, content)
}

func (sc SaoClient) CreateFile(ctx context.Context, orderMeta types.OrderMeta) (apitypes.CreateResp, error) {
	return sc.gatewayApi.CreateFile(ctx, orderMeta)
}

func (sc SaoClient) Load(ctx context.Context, orderMeta types.OrderMeta) (apitypes.LoadResp, error) {
	return sc.gatewayApi.Load(ctx, orderMeta)
}

func (sc SaoClient) Delete(ctx context.Context, owner string, key string, group string) (apitypes.DeleteResp, error) {
	return sc.gatewayApi.Delete(ctx, owner, key, group)
}

func (sc SaoClient) ShowCommits(ctx context.Context, owner string, key string, group string) (apitypes.ShowCommitsResp, error) {
	return sc.gatewayApi.ShowCommits(ctx, owner, key, group)
}

func (sc SaoClient) Update(ctx context.Context, orderProposal types.ClientOrderProposal, orderMeta types.OrderMeta, patch []byte) (apitypes.UpdateResp, error) {
	return sc.gatewayApi.Update(ctx, orderProposal, orderMeta, patch)
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
