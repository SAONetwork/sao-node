package chain

import (
	"context"
	"sao-node/types"

	sdkquerytypes "github.com/cosmos/cosmos-sdk/types/query"

	modeltypes "github.com/SaoNetwork/sao/x/model/types"

	saotypes "github.com/SaoNetwork/sao/x/sao/types"
)

func (c *ChainSvc) ListMeta(ctx context.Context, offset uint64, limit uint64) ([]modeltypes.Metadata, uint64, error) {
	resp, err := c.modelClient.MetadataAll(ctx, &modeltypes.QueryAllMetadataRequest{
		Pagination: &sdkquerytypes.PageRequest{Offset: offset, Limit: limit, Reverse: true}})
	if err != nil {
		return make([]modeltypes.Metadata, 0), 0, types.Wrap(types.ErrQueryNodeFailed, err)
	}
	return resp.Metadata, resp.Pagination.Total, nil
}

func (c *ChainSvc) GetMeta(ctx context.Context, dataId string) (*modeltypes.QueryGetMetadataResponse, error) {
	resp, err := c.modelClient.Metadata(ctx, &modeltypes.QueryGetMetadataRequest{
		DataId: dataId,
	})
	if err != nil {
		return nil, err
	}
	return resp, nil
}

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
		return nil, types.Wrap(types.ErrQueryMetadataFailed, err)
	}
	return resp, nil
}

func (c *ChainSvc) UpdatePermission(ctx context.Context, signer string, proposal *types.PermissionProposal) (string, error) {
	signerAcc, err := c.cosmos.Account(signer)
	if err != nil {
		return "", types.Wrap(types.ErrAccountNotFound, err)
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
		return "", types.Wrap(types.ErrTxProcessFailed, err)
	}
	// log.Debug("MsgStore result: ", txResp)
	if txResp.TxResponse.Code != 0 {
		return "", types.Wrapf(types.ErrTxProcessFailed, "MsgUpdataPermission tx hash=%s, code=%d", txResp.TxResponse.TxHash, txResp.TxResponse.Code)
	}

	return txResp.TxResponse.TxHash, nil
}
