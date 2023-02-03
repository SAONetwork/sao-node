package chain

import (
	"context"
	"sao-node/types"

	saotypes "github.com/SaoNetwork/sao/x/sao/types"
	"golang.org/x/xerrors"
)

func (c *ChainSvc) QueryMetadata(ctx context.Context, req *types.MetadataProposal, height int64) (*saotypes.QueryMetadataResponse, error) {
	clientctx := c.cosmos.Context()
	if height > 0 {
		clientctx = clientctx.WithHeight(height)
	}
	saoClient := saotypes.NewQueryClient(clientctx)
	resp, err := saoClient.Metadata(ctx, &saotypes.QueryMetadataRequest{
		Proposal: saotypes.QueryProposal{
			Owner:           req.Proposal.Owner,
			Keyword:         req.Proposal.Keyword,
			GroupId:         req.Proposal.GroupId,
			KeywordType:     uint32(req.Proposal.KeywordType),
			LastValidHeight: req.Proposal.LastValidHeight,
			Gateway:         req.Proposal.Gateway,
			CommitId:        req.Proposal.CommitId,
			Version:         req.Proposal.Version,
		},
		JwsSignature: saotypes.JwsSignature{
			Protected: req.JwsSignature.Protected,
			Signature: req.JwsSignature.Signature,
		},
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
		return "", xerrors.Errorf("MsgUpdataPermission tx %v failed: code=%d", txResp.TxResponse.TxHash, txResp.TxResponse.Code)
	}

	return txResp.TxResponse.TxHash, nil
}
