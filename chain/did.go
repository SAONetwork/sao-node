package chain

import (
	"context"

	"github.com/SaoNetwork/sao-node/types"

	"golang.org/x/xerrors"

	saodid "github.com/SaoNetwork/sao-did"
	"github.com/SaoNetwork/sao-did/parser"
	"github.com/SaoNetwork/sao-did/sid"
	saodidtypes "github.com/SaoNetwork/sao-did/types"

	sidtypes "github.com/SaoNetwork/sao/x/did/types"
)

func (c *ChainSvc) GetSidDocument(ctx context.Context, versionId string) (*sid.SidDocument, error) {
	resp, err := c.didClient.SidDocument(ctx, &sidtypes.QueryGetSidDocumentRequest{VersionId: versionId})
	if err != nil {
		return nil, types.Wrap(types.ErrGetSidDocumentFailed, err)
	}
	if resp.SidDocument.VersionId == "" {
		return nil, nil
	}
	var keys = make([]*sid.PubKey, 0)
	for _, pk := range resp.SidDocument.Keys {
		keys = append(keys, &sid.PubKey{
			Name:  pk.Name,
			Value: pk.Value,
		})
	}

	return &sid.SidDocument{
		VersionId: resp.SidDocument.VersionId,
		Keys:      keys,
	}, nil
}

func (c *ChainSvc) UpdateDidBinding(ctx context.Context, creator string, did string, accountId string) (string, error) {
	msg := &sidtypes.MsgUpdatePaymentAddress{
		Creator:   creator,
		Did:       did,
		AccountId: accountId,
	}
	resultChan := make(chan BroadcastTxJobResult)
	c.broadcastMsg(creator, msg, resultChan)
	result := <-resultChan
	if result.err != nil {
		return "", types.Wrap(types.ErrTxProcessFailed, result.err)
	}
	if result.resp.TxResponse.Code != 0 {
		return "", types.Wrapf(types.ErrTxProcessFailed, "MsgUpdatePaymentAddress tx hash=%s, code=%d", result.resp.TxResponse.TxHash, result.resp.TxResponse.Code)
	}
	return result.resp.TxResponse.TxHash, nil
}

func (c *ChainSvc) QueryPaymentAddress(ctx context.Context, did string) (string, error) {
	msg := &sidtypes.QueryGetPaymentAddressRequest{
		Did: did,
	}
	paymentAddrResp, err := c.didClient.PaymentAddress(ctx, msg)
	if err != nil {
		return "", err
	}
	return paymentAddrResp.PaymentAddress.Address, nil
}

func (c *ChainSvc) GetDidInfo(ctx context.Context, did string) (types.DidInfo, error) {
	_, err := c.didClient.ValidateDid(ctx, &sidtypes.QueryValidateDidRequest{
		Did: did,
	})
	if err != nil {
		return nil, err
	}

	paymentAddressResp, err := c.didClient.PaymentAddress(ctx, &sidtypes.QueryGetPaymentAddressRequest{
		Did: did,
	})
	if err != nil {
		return nil, err
	}

	getSidDocFunc := func(versionId string) (*sid.SidDocument, error) {
		return c.GetSidDocument(ctx, versionId)
	}

	getDidResolutionFunc := func(did string) (saodidtypes.DidResolutionResult, error) {
		didManager, err := saodid.NewDidManagerWithDid(did, getSidDocFunc)
		if err != nil {
			return saodidtypes.DidResolutionResult{}, err
		}
		result := didManager.Resolver.Resolve(did, saodidtypes.DidResolutionOptions{})
		return result, nil
	}

	pd, err := parser.Parse(did)
	if err != nil {
		return nil, err
	}

	if pd.Method == "sid" {
		var info types.SidInfo
		info.Did = did
		info.PaymentAddress = paymentAddressResp.PaymentAddress.Address
		accountAuthsResp, err := c.didClient.GetAllAccountAuths(ctx, &sidtypes.QueryGetAllAccountAuthsRequest{
			Did: did,
		})
		if err != nil {
			return nil, err
		}
		for _, accAuth := range accountAuthsResp.AccountAuths {
			accountIdResp, err := c.didClient.AccountId(ctx, &sidtypes.QueryGetAccountIdRequest{
				AccountDid: accAuth.AccountDid,
			})
			if err != nil {
				return nil, err
			}
			info.Accounts = append(info.Accounts, types.Account{
				AccountId:            accountIdResp.AccountId.AccountId,
				AccountDid:           accAuth.AccountDid,
				AccountEncryptedSeed: accAuth.AccountEncryptedSeed,
				SidEncryptedAccount:  accAuth.SidEncryptedAccount,
			})
		}

		pastSeedsResp, err := c.didClient.PastSeeds(ctx, &sidtypes.QueryGetPastSeedsRequest{
			Did: did,
		})
		if err == nil {
			info.PastSeeds = pastSeedsResp.PastSeeds.Seeds
		}

		versionsResp, err := c.didClient.SidDocumentVersion(ctx, &sidtypes.QueryGetSidDocumentVersionRequest{
			DocId: pd.ID,
		})
		if err != nil {
			return nil, err
		}

		for _, version := range versionsResp.SidDocumentVersion.VersionList {
			didResolution, err := getDidResolutionFunc("did:sid:" + pd.ID + "?versionId=" + version)
			if err != nil {
				return nil, err
			}
			if didResolution.DidResolutionMetadata.Error != "" {
				return nil, xerrors.New(didResolution.DidResolutionMetadata.Error)
			}
			info.DidDocuments = append(info.DidDocuments, types.DidDocument{
				Version:  version,
				Document: didResolution.DidDocument,
			})
		}

		return info, nil
	} else if pd.Method == "key" {
		var info types.KidInfo
		info.Did = did
		info.PaymentAddress = paymentAddressResp.PaymentAddress.Address
		didResolution, err := getDidResolutionFunc(did)
		if err != nil {
			return nil, err
		}
		if didResolution.DidResolutionMetadata.Error != "" {
			return nil, xerrors.New(didResolution.DidResolutionMetadata.Error)
		}
		info.Document = didResolution.DidDocument

		return info, nil
	} else {
		return nil, xerrors.New("Unsupported did type")
	}
}
