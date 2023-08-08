package chain

import (
	"context"
	"github.com/SaoNetwork/sao-node/types"

	modeltypes "github.com/SaoNetwork/sao/x/model/types"
	sdkquerytypes "github.com/cosmos/cosmos-sdk/types/query"

	saotypes "github.com/SaoNetwork/sao/x/sao/types"
)

func (c *ChainSvc) ListMeta(ctx context.Context, offset uint64, limit uint64) ([]modeltypes.Metadata, uint64, error) {
	resp, err := c.modelClient.MetadataAll(ctx, &modeltypes.QueryAllMetadataRequest{
		Pagination: &sdkquerytypes.PageRequest{Offset: offset, Limit: limit, Reverse: false, CountTotal: true}})
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

func (c *ChainSvc) GetModel(ctx context.Context, key string) (*modeltypes.QueryGetModelResponse, error) {
	resp, err := c.modelClient.Model(ctx, &modeltypes.QueryGetModelRequest{
		Key: key,
	})
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *ChainSvc) QueryMetadata(ctx context.Context, req *types.MetadataProposal, height int64, modelClient bool) (*saotypes.QueryMetadataResponse, error) {
	clientctx := c.cosmos.Context()
	if req.Proposal.KeywordType <= 1 && modelClient {
		modelClient := modeltypes.NewQueryClient(clientctx)
		resp, err := modelClient.Metadata(ctx, &modeltypes.QueryGetMetadataRequest{
			DataId: req.Proposal.Keyword,
		})
		if err != nil {
			return nil, types.Wrap(types.ErrQueryMetadataFailed, err)
		}
		shards := make(map[string]*saotypes.ShardMeta)
		for k, v := range resp.Shards {
			shards[k] = &saotypes.ShardMeta{
				ShardId: v.ShardId,
				Peer:    v.Peer,
				Cid:     v.Cid,
				// Provider and Sp fields are not available in modeltypes.ShardMeta
			}
		}
		return &saotypes.QueryMetadataResponse{
			Metadata: saotypes.Metadata{
				DataId:         resp.Metadata.DataId,
				Owner:          resp.Metadata.Owner,
				Alias:          resp.Metadata.Alias,
				GroupId:        resp.Metadata.GroupId,
				OrderId:        resp.Metadata.OrderId,
				Tags:           resp.Metadata.Tags,
				Cid:            resp.Metadata.Cid,
				Commits:        resp.Metadata.Commits,
				ExtendInfo:     resp.Metadata.ExtendInfo,
				Update:         resp.Metadata.Update,
				Commit:         resp.Metadata.Commit,
				Rule:           resp.Metadata.Rule,
				Duration:       resp.Metadata.Duration,
				CreatedAt:      resp.Metadata.CreatedAt,
				Status:         resp.Metadata.Status,
			},
			Shards: shards,
		}, nil
	} else {
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
}

func (c *ChainSvc) UpdatePermission(ctx context.Context, signer string, proposal *types.PermissionProposal) (string, error) {
	txAddress := signer
	defer func() {
		if c.ap != nil && txAddress != signer {
			c.ap.SetAddressAvailable(txAddress)
		}
	}()

	var err error
	if c.ap != nil {
		txAddress, err = c.ap.GetRandomAddress(ctx)
		if err != nil {
			return "", types.Wrap(types.ErrAccountNotFound, err)
		}

		_, err = c.cosmos.Account(txAddress)
		if err != nil {
			return "", types.Wrap(types.ErrAccountNotFound, err)
		}
	}

	// TODO: Cid
	msg := &saotypes.MsgUpdataPermission{
		Creator:  txAddress,
		Proposal: proposal.Proposal,
		JwsSignature: saotypes.JwsSignature{
			Protected: proposal.JwsSignature.Protected,
			Signature: proposal.JwsSignature.Signature,
		},
		Provider: signer,
	}

	resultChan := make(chan BroadcastTxJobResult)
	c.broadcastMsg(txAddress, msg, resultChan)
	result := <-resultChan
	if result.err != nil {
		return "", types.Wrap(types.ErrTxProcessFailed, result.err)
	}
	// log.Debug("MsgStore result: ", txResp)
	if result.resp.TxResponse.Code != 0 {
		return "", types.Wrapf(types.ErrTxProcessFailed, "MsgUpdataPermission tx hash=%s, code=%d", result.resp.TxResponse.TxHash, result.resp.TxResponse.Code)
	}

	return result.resp.TxResponse.TxHash, nil
}

func (c *ChainSvc) ListMetaByDid(ctx context.Context, did string) ([]modeltypes.Metadata, error) {

	var offset uint64 = 0
	var limit uint64 = 100
	allMetadatas := []modeltypes.Metadata{}
	for {
		metaList, total, err := c.ListMeta(ctx, offset, limit)
		if err != nil {
			return nil, err
		}

		for _, meta := range metaList {
			if meta.Owner == did {
				allMetadatas = append(allMetadatas, meta)
			}
		}

		if offset+limit <= total {
			offset += limit
		} else {
			break
		}
	}
	return allMetadatas, nil
}
