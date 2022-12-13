package chain

import (
	"context"

	"golang.org/x/xerrors"

	"github.com/SaoNetwork/sao/x/did/types"
)

func (c *ChainSvc) GetSidDocument(ctx context.Context, versionId string) (*types.SidDocument, error) {
	resp, err := c.didClient.SidDocument(ctx, &types.QueryGetSidDocumentRequest{VersionId: versionId})
	if err != nil {
		return nil, err
	}
	if resp.SidDocument.VersionId == "" {
		return nil, nil
	}
	return &resp.SidDocument, nil
}

func (c *ChainSvc) UpdateDidBinding(ctx context.Context, creator string, did string, accountId string) (string, error) {
	signerAcc, err := c.cosmos.Account(creator)
	if err != nil {
		return "", xerrors.Errorf("chain get account: %w, check the keyring please", err)
	}

	msg := &types.MsgUpdatePaymentAddress{
		Creator:   creator,
		Did:       did,
		AccountId: accountId,
	}
	txResp, err := c.cosmos.BroadcastTx(ctx, signerAcc, msg)
	if err != nil {
		return "", err
	}
	if txResp.TxResponse.Code != 0 {
		return "", xerrors.Errorf("MsgComplete tx %v failed: code=%d", txResp.TxResponse.TxHash, txResp.TxResponse.Code)
	}
	return txResp.TxResponse.TxHash, nil
}
