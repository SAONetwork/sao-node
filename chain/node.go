package chain

import (
	"context"
	"fmt"
	"time"

	nodetypes "github.com/SaoNetwork/sao/x/node/types"
	"golang.org/x/xerrors"
)

func (c *ChainSvc) Login(ctx context.Context, creator string) (string, error) {
	account, err := c.cosmos.Account(creator)
	if err != nil {
		return "", xerrors.Errorf("chain get account: %w, check the keyring please", err)
	}

	msg := &nodetypes.MsgLogin{
		Creator: creator,
	}

	txResp, err := c.cosmos.BroadcastTx(ctx, account, msg)
	if err != nil {
		return "", err
	}
	if txResp.TxResponse.Code != 0 {
		return "", xerrors.Errorf("MsgLogin tx %v failed: code=%d", txResp.TxResponse.TxHash, txResp.TxResponse.Code)
	}
	return txResp.TxResponse.TxHash, nil
}

func (c *ChainSvc) Logout(ctx context.Context, creator string) (string, error) {
	account, err := c.cosmos.Account(creator)
	if err != nil {
		return "", xerrors.Errorf("chain get account: %w, check the keyring please", err)
	}

	msg := &nodetypes.MsgLogout{
		Creator: creator,
	}
	txResp, err := c.cosmos.BroadcastTx(ctx, account, msg)
	if err != nil {
		return "", err
	}

	if txResp.TxResponse.Code != 0 {
		return "", xerrors.Errorf("MsgLogout tx %v failed: code=%d", txResp.TxResponse.TxHash, txResp.TxResponse.Code)
	}
	return txResp.TxResponse.TxHash, nil
}

func (c *ChainSvc) Reset(ctx context.Context, creator string, peerInfo string, status uint32) (string, error) {
	account, err := c.cosmos.Account(creator)
	if err != nil {
		return "", xerrors.Errorf("chain get account: %w, check the keyring please", err)
	}

	msg := &nodetypes.MsgReset{
		Creator: creator,
		Peer:    peerInfo,
		Status:  status,
	}
	txResp, err := c.cosmos.BroadcastTx(ctx, account, msg)
	if err != nil {
		return "", err
	}
	if txResp.TxResponse.Code != 0 {
		return "", xerrors.Errorf("MsgReset tx %v failed: code=%d", txResp.TxResponse.TxHash, txResp.TxResponse.Code)
	}
	return txResp.TxResponse.TxHash, nil
}

func (c *ChainSvc) ClaimReward(ctx context.Context, creator string) (string, error) {
	account, err := c.cosmos.Account(creator)
	if err != nil {
		return "", xerrors.Errorf("chain get account: %w, check the keyring please", err)
	}

	msg := &nodetypes.MsgClaimReward{
		Creator: creator,
	}
	txResp, err := c.cosmos.BroadcastTx(ctx, account, msg)
	if err != nil {
		return "", err
	}
	if txResp.TxResponse.Code != 0 {
		return "", xerrors.Errorf("MsgClaimReward tx %v failed: code=%d", txResp.TxResponse.TxHash, txResp.TxResponse.Code)
	}
	return txResp.TxResponse.TxHash, nil
}

func (c *ChainSvc) GetNodePeer(ctx context.Context, creator string) (string, error) {
	resp, err := c.nodeClient.Node(ctx, &nodetypes.QueryGetNodeRequest{
		Creator: creator,
	})
	if err != nil {
		return "", err
	}
	return resp.Node.Peer, nil
}

func (c *ChainSvc) GetNodeStatus(ctx context.Context, creator string) (uint32, error) {
	resp, err := c.nodeClient.Node(ctx, &nodetypes.QueryGetNodeRequest{
		Creator: creator,
	})
	if err != nil {
		return 0, err
	}
	return resp.Node.Status, nil
}

func (c *ChainSvc) ShowNodeInfo(ctx context.Context, creator string) {
	resp, err := c.nodeClient.Node(ctx, &nodetypes.QueryGetNodeRequest{
		Creator: creator,
	})
	if err != nil {
		log.Error(err.Error())
		return
	}
	fmt.Printf("Node Information:%+v\n", resp.Node)

	pledgeResp, err := c.nodeClient.Pledge(ctx, &nodetypes.QueryGetPledgeRequest{
		Creator: creator,
	})
	if err != nil {
		log.Error(err.Error())
		return
	}
	fmt.Printf("Node Pledge:%+v\n", pledgeResp.Pledge)
}

func (c *ChainSvc) StartStatusReporter(ctx context.Context, creator string, status uint32) {
	go func() {
		ticker := time.NewTicker(15 * time.Minute)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				txHash, err := c.Reset(ctx, creator, "", status)
				if err != nil {
					log.Error(err.Error())
				}

				log.Infof("Reported node status[%b] to SAO network, txHash=%s", status, txHash)
			case <-ctx.Done():
				return
			}
		}
	}()
}
