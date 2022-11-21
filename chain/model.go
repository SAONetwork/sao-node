package chain

import (
	"context"

	modeltypes "github.com/SaoNetwork/sao/x/model/types"
	"golang.org/x/xerrors"
)

func (c *ChainSvc) QueryMeta(ctx context.Context, dataId string, height int64) (*modeltypes.QueryGetMetadataResponse, error) {
	clientctx := c.cosmos.Context()
	if height > 0 {
		clientctx = clientctx.WithHeight(height)
	}
	modelClient := modeltypes.NewQueryClient(clientctx)
	queryResp, err := modelClient.Metadata(ctx, &modeltypes.QueryGetMetadataRequest{
		DataId: dataId,
	})
	if err != nil {
		return nil, xerrors.Errorf("QueryMeta failed, " + err.Error())
	}
	return queryResp, nil
}

func (c *ChainSvc) QueryDataId(ctx context.Context, key string) (string, error) {
	queryResp, err := c.modelClient.Model(ctx, &modeltypes.QueryGetModelRequest{
		Key: key,
	})
	if err != nil {
		return "", xerrors.Errorf("QueryDataId failed, " + err.Error())
	}
	return queryResp.Model.Data, nil
}
