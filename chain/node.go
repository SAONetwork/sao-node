package chain

import (
	"context"
	"fmt"
	"sao-node/types"
	"strings"
	"time"

	nodetypes "github.com/SaoNetwork/sao/x/node/types"
)

func (c *ChainSvc) Create(ctx context.Context, creator string) (string, error) {
	msg := &nodetypes.MsgCreate{
		Creator: creator,
	}

	resultChan := make(chan BroadcastTxJobResult)
	c.broadcastMsg(creator, msg, resultChan)
	result := <-resultChan
	if result.err != nil {
		return "", types.Wrap(types.ErrTxProcessFailed, result.err)
	}
	if result.resp.TxResponse.Code != 0 {
		return "", types.Wrapf(types.ErrTxProcessFailed, "MsgCreate tx hash=%s, code=%d", result.resp.TxResponse.TxHash, result.resp.TxResponse.Code)
	}
	return result.resp.TxResponse.TxHash, nil
}

func (c *ChainSvc) Reset(ctx context.Context, creator string, peerInfo string, status uint32) (string, error) {
	msg := &nodetypes.MsgReset{
		Creator: creator,
		Peer:    peerInfo,
		Status:  status,
	}
	resultChan := make(chan BroadcastTxJobResult)
	c.broadcastMsg(creator, msg, resultChan)
	result := <-resultChan
	if result.err != nil {
		return "", types.Wrap(types.ErrTxProcessFailed, result.err)
	}
	if result.resp.TxResponse.Code != 0 {
		return "", types.Wrapf(types.ErrTxProcessFailed, "MsgReset tx hash=%s, code=%d", result.resp.TxHash, result.resp.TxResponse.Code)
	}
	return result.resp.TxResponse.TxHash, nil
}

func (c *ChainSvc) ClaimReward(ctx context.Context, creator string) (string, error) {
	msg := &nodetypes.MsgClaimReward{
		Creator: creator,
	}
	resultChan := make(chan BroadcastTxJobResult)
	c.broadcastMsg(creator, msg, resultChan)
	result := <-resultChan
	if result.err != nil {
		return "", types.Wrap(types.ErrTxProcessFailed, result.err)
	}
	if result.resp.TxResponse.Code != 0 {
		return "", types.Wrapf(types.ErrTxProcessFailed, "MsgClaimReward tx hash=%s, code=%d", result.resp.TxResponse.TxHash, result.resp.TxResponse.Code)
	}
	return result.resp.TxResponse.TxHash, nil
}

func (c *ChainSvc) GetNodePeer(ctx context.Context, creator string) (string, error) {
	resp, err := c.nodeClient.Node(ctx, &nodetypes.QueryGetNodeRequest{
		Creator: creator,
	})
	if err != nil {
		fmt.Println("creator:", creator, err)
		return "", types.Wrap(types.ErrQueryNodeFailed, err)
	}
	return resp.Node.Peer, nil
}

func (c *ChainSvc) GetNodeStatus(ctx context.Context, creator string) (uint32, error) {
	resp, err := c.nodeClient.Node(ctx, &nodetypes.QueryGetNodeRequest{
		Creator: creator,
	})
	if err != nil {
		fmt.Println("creator:", creator, err)
		return 0, types.Wrap(types.ErrQueryNodeFailed, err)
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
	fmt.Println("Node Information")
	fmt.Println("Creator:", resp.Node.Creator)
	fmt.Printf("Status:%b\n", resp.Node.Status)
	fmt.Println("Reputation:", resp.Node.Reputation)
	fmt.Println("LastAliveHeight:", resp.Node.LastAliveHeight)
	for _, peer := range strings.Split(resp.Node.Peer, ",") {
		fmt.Println("P2P Peer Info:", peer)
	}

	pledgeResp, err := c.nodeClient.Pledge(ctx, &nodetypes.QueryGetPledgeRequest{
		Creator: creator,
	})
	if err != nil {
		fmt.Println("No Pledge Info")
		return
	} else {
		fmt.Println("Node Pledge")
		fmt.Println("Reward:", pledgeResp.Pledge.Reward)
		fmt.Println("Reward Debt:", pledgeResp.Pledge.RewardDebt)
		fmt.Println("TotalOrderPledged:", pledgeResp.Pledge.TotalOrderPledged)
		fmt.Println("TotalStoragePledged:", pledgeResp.Pledge.TotalStoragePledged)
		fmt.Println("TotalStorage:", pledgeResp.Pledge.TotalStorage)
		fmt.Println("LastRewardAt:", pledgeResp.Pledge.LastRewardAt)
	}
}

func (c *ChainSvc) ListNodes(ctx context.Context) ([]nodetypes.Node, error) {
	resp, err := c.nodeClient.NodeAll(ctx, &nodetypes.QueryAllNodeRequest{Status: 0})
	if err != nil {
		return make([]nodetypes.Node, 0), types.Wrap(types.ErrQueryNodeFailed, err)
	}
	return resp.Node, nil
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
