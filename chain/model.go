package chain

import (
	"context"
	"sao-storage-node/types"

	modeltypes "github.com/SaoNetwork/sao/x/model/types"
	saotypes "github.com/SaoNetwork/sao/x/sao/types"
	"golang.org/x/xerrors"
)

func (c *ChainSvc) Que3ryMeta(ctx context.Context, dataId string, height int64) (*modeltypes.QueryGetMetadataResponse, error) {
	return nil, xerrors.Errorf("Invalid query...")
}

func (c *ChainSvc) QueryDataId(ctx context.Context, key string) (string, error) {
	return "", xerrors.Errorf("Invalid query...")
}

func (c *ChainSvc) QueryMetadata(ctx context.Context, req *types.MetadataProposal, height int64) (*saotypes.QueryMetadataResponse, error) {
	clientctx := c.cosmos.Context()
	if height > 0 {
		clientctx = clientctx.WithHeight(height)
	}
	saoClient := saotypes.NewQueryClient(clientctx)
	resp, err := saoClient.Metadata(ctx, &saotypes.QueryMetadataRequest{
		Proposal:     req.Proposal,
		JwsSignature: req.JwsSignature,
	})
	if err != nil {
		return nil, xerrors.Errorf("QueryMetadata failed, " + err.Error())
	}
	return resp, nil
}
