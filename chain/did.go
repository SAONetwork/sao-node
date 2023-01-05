package chain

import (
	"context"
	"fmt"

	saodid "github.com/SaoNetwork/sao-did"
	sid "github.com/SaoNetwork/sao-did/sid"
	saodidtypes "github.com/SaoNetwork/sao-did/types"
	"golang.org/x/xerrors"

	"github.com/SaoNetwork/sao/x/did/types"
)

func (c *ChainSvc) GetSidDocument(ctx context.Context, versionId string) (*sid.SidDocument, error) {
	resp, err := c.didClient.SidDocument(ctx, &types.QueryGetSidDocumentRequest{VersionId: versionId})
	if err != nil {
		return nil, err
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

func (c *ChainSvc) ShowDidInfo(ctx context.Context, accountDid string) {
	_, err := c.didClient.ValidateDid(ctx, &types.QueryValidateDidRequest{
		Did: accountDid,
	})
	if err != nil {
		log.Error(err.Error())
		return
	}

	resp, err := c.didClient.AccountAuth(ctx, &types.QueryGetAccountAuthRequest{
		AccountDid: accountDid,
	})
	if err != nil {
		log.Error(err.Error())
		return
	}
	fmt.Println("AccountAuth:", resp)

	pastSeedsResp, err := c.didClient.PastSeeds(ctx, &types.QueryGetPastSeedsRequest{
		Did: accountDid,
	})
	if err != nil {
		log.Error(err.Error())
		return
	}
	fmt.Println("PastSeeds:", pastSeedsResp)

	didBindingProofResp, err := c.didClient.DidBindingProof(ctx, &types.QueryGetDidBindingProofRequest{
		AccountId: accountDid,
	})
	if err != nil {
		log.Error(err.Error())
		return
	}
	fmt.Println("DidBindingProof:", didBindingProofResp)

	paymentAddressResp, err := c.didClient.PaymentAddress(ctx, &types.QueryGetPaymentAddressRequest{
		Did: accountDid,
	})
	if err != nil {
		log.Error(err.Error())
		return
	}
	fmt.Println("PaymentAddress:", paymentAddressResp)

	getSidDocFunc := func(versionId string) (*sid.SidDocument, error) {
		return c.GetSidDocument(ctx, versionId)
	}

	didManager, err := saodid.NewDidManagerWithDid(accountDid, getSidDocFunc)
	if err != nil {
		log.Error(err.Error())
		return
	}
	result := didManager.Resolver.Resolve(accountDid, saodidtypes.DidResolutionOptions{})
	fmt.Println("DidResolution:", result)
}
