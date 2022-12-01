package chain

import (
	"context"
	"github.com/SaoNetwork/sao/x/did/types"
)

func (c *ChainSvc) GetSidDocument(ctx context.Context, versionId string) (*types.SidDocument, error) {
	resp, err := c.didClient.SidDocument(ctx, &types.QueryGetSidDocumentRequest{VersionId: versionId})
	if err != nil {
		return nil, err
	}
	emptyDoc := types.SidDocument{}
	if resp.SidDocument == emptyDoc {
		return nil, nil
	}
	return &resp.SidDocument, nil
}
