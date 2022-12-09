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

func (c *ChainSvc) UpdatePermission(ctx context.Context, signer string, proposal *types.PermissionProposal) (string, error) {
	signerAcc, err := c.cosmos.Account(signer)
	if err != nil {
		return "", xerrors.Errorf("%w, check the keyring please", err)
	}

	// TODO: Cid
	msg := &saotypes.MsgUpdataPermission{
		Creator:  signer,
		Proposal: proposal.Proposal,
		JwsSignature: saotypes.JwsSignature{
			Protected: proposal.JwsSignature.Protected,
			Signature: proposal.JwsSignature.Signature,
		},
	}

	txResp, err := c.cosmos.BroadcastTx(ctx, signerAcc, msg)
	if err != nil {
		return "", err
	}
	// log.Debug("MsgStore result: ", txResp)
	if txResp.TxResponse.Code != 0 {
		return "", xerrors.Errorf("MsgStore tx %v failed: code=%d", txResp.TxResponse.TxHash, txResp.TxResponse.Code)
	}
	var resp saotypes.MsgUpdataPermissionResponse
	err = txResp.Decode(&resp)
	if err != nil {
		return "", err
	}
	return txResp.TxResponse.TxHash, nil
}
