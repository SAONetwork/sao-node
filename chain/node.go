package chain

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/SaoNetwork/sao-node/types"

	nodetypes "github.com/SaoNetwork/sao/x/node/types"
	saotypes "github.com/SaoNetwork/sao/x/sao/types"
	sdktypes "github.com/cosmos/cosmos-sdk/types"
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

func (c *ChainSvc) Reset(ctx context.Context, creator string, peerInfo string, status uint32,
	txAddresses []string, description *nodetypes.Description) (string, error) {
	_, err := c.cosmos.Account(creator)
	if err != nil {
		return "", types.Wrap(types.ErrAccountNotFound, err)
	}

	msg := &nodetypes.MsgReset{
		Creator:     creator,
		Peer:        peerInfo,
		Status:      status,
		TxAddresses: txAddresses,
		Description: description,
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
	var claimResp nodetypes.MsgClaimRewardResponse
	err := result.resp.Decode(&claimResp)
	if err != nil {
		fmt.Println("decode claim resp err: ", err)
	} else {
		fmt.Println("total claim:", claimResp.ClaimedReward)
	}
	return result.resp.TxResponse.TxHash, nil
}

func (c *ChainSvc) GetFault(ctx context.Context, faultId string) (*nodetypes.Fault, error) {
	resp, err := c.nodeClient.Fault(ctx, &nodetypes.QueryFaultRequest{FaultId: faultId})
	if err != nil {
		return nil, types.Wrap(types.ErrQueryFaultFailed, err)
	}

	return resp.Fault, nil
}

func (c *ChainSvc) GetMyFaults(ctx context.Context, provider string) ([]string, error) {
	resp, err := c.nodeClient.AllFaults(ctx, &nodetypes.QueryAllFaultsRequest{Provider: provider})
	if err != nil {
		return nil, types.Wrap(types.ErrQueryFaultsFailed, err)
	}

	return resp.FaultIds, nil
}

func (c *ChainSvc) ReportFaults(ctx context.Context, creator string, provider string, faults []*saotypes.Fault) ([]string, error) {
	msg := &saotypes.MsgReportFaults{
		Creator:  creator,
		Provider: provider,
		Faults:   faults,
	}
	resultChan := make(chan BroadcastTxJobResult)
	c.broadcastMsg(creator, msg, resultChan)
	result := <-resultChan
	if result.err != nil {
		return nil, types.Wrap(types.ErrTxProcessFailed, result.err)
	}
	if result.resp.TxResponse.Code != 0 {
		return nil, types.Wrapf(types.ErrTxProcessFailed, "MsgReportFaults tx hash=%s, code=%d", result.resp.TxResponse.TxHash, result.resp.TxResponse.Code)
	}
	var resp saotypes.MsgReportFaultsResponse
	err := result.resp.Decode(&resp)
	if err != nil {
		return nil, types.Wrapf(types.ErrTxProcessFailed, "failed to decode MsgReportFaultsResponse, due to %v", err)
	}

	return resp.FaultIds, nil
}

func (c *ChainSvc) RecoverFaults(ctx context.Context, creator string, provider string, faults []*saotypes.Fault) ([]string, error) {
	msg := &saotypes.MsgRecoverFaults{
		Creator:  creator,
		Provider: provider,
		Faults:   faults,
	}
	resultChan := make(chan BroadcastTxJobResult)
	c.broadcastMsg(creator, msg, resultChan)
	result := <-resultChan
	if result.err != nil {
		return nil, types.Wrap(types.ErrTxProcessFailed, result.err)
	}
	if result.resp.TxResponse.Code != 0 {
		return nil, types.Wrapf(types.ErrTxProcessFailed, "MsgRecoverFaults tx hash=%s, code=%d", result.resp.TxResponse.TxHash, result.resp.TxResponse.Code)
	}
	return nil, nil
}

func (c *ChainSvc) AddVstorage(ctx context.Context, creator string, size uint64) (string, error) {
	_, err := c.cosmos.Account(creator)
	if err != nil {
		return "", types.Wrap(types.ErrAccountNotFound, err)
	}

	msg := &nodetypes.MsgAddVstorage{
		Creator: creator,
		Size_:   size,
	}
	resultChan := make(chan BroadcastTxJobResult)
	c.broadcastMsg(creator, msg, resultChan)
	result := <-resultChan
	if result.err != nil {
		return "", types.Wrap(types.ErrTxProcessFailed, result.err)
	}
	if result.resp.TxResponse.Code != 0 {
		return "", types.Wrapf(types.ErrTxProcessFailed, "MsgAddVstorage tx hash=%s, code=%d", result.resp.TxHash, result.resp.TxResponse.Code)
	}
	return result.resp.TxResponse.TxHash, nil
}

func (c *ChainSvc) RemoveVstorage(ctx context.Context, creator string, size uint64) (string, error) {
	_, err := c.cosmos.Account(creator)
	if err != nil {
		return "", types.Wrap(types.ErrAccountNotFound, err)
	}

	msg := &nodetypes.MsgRemoveVstorage{
		Creator: creator,
		Size_:   size,
	}
	resultChan := make(chan BroadcastTxJobResult)
	c.broadcastMsg(creator, msg, resultChan)
	result := <-resultChan
	if result.err != nil {
		return "", types.Wrap(types.ErrTxProcessFailed, result.err)
	}
	if result.resp.TxResponse.Code != 0 {
		return "", types.Wrapf(types.ErrTxProcessFailed, "MsgRemoveVstorage tx hash=%s, code=%d", result.resp.TxHash, result.resp.TxResponse.Code)
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
		fmt.Println("TotalStoragePledged:", pledgeResp.Pledge.TotalStoragePledged)
		fmt.Println("TotalStorage:", pledgeResp.Pledge.TotalStorage)
	}
}

func (c *ChainSvc) ListNodes(ctx context.Context) ([]nodetypes.Node, error) {
	resp, err := c.nodeClient.NodeAll(ctx, &nodetypes.QueryAllNodeRequest{Status: 0})
	if err != nil {
		return make([]nodetypes.Node, 0), types.Wrap(types.ErrQueryNodeFailed, err)
	}
	return resp.Node, nil
}

func (c *ChainSvc) GetPledgeInfo(ctx context.Context, creator string) (*sdktypes.Coin, error) {
	pledgeResp, err := c.nodeClient.Pledge(ctx, &nodetypes.QueryGetPledgeRequest{
		Creator: creator,
	})
	if err != nil {
		return nil, types.Wrap(types.ErrQueryPledgeFailed, err)
	}
	return &pledgeResp.Pledge.TotalStoragePledged, nil
}

func (c *ChainSvc) StartStatusReporter(ctx context.Context, creator string, status uint32) {
	go func() {
		ticker := time.NewTicker(15 * time.Minute)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				txHash, err := c.Reset(ctx, creator, "", status, make([]string, 0), nil)
				if err != nil {
					log.Error(err.Error())
				}

				faultIds, err := c.GetMyFaults(ctx, creator)
				if err != nil {
					log.Error(err.Error())
				} else {
					for _, faultId := range faultIds {
						log.Errorf("!!!STORAGE FAULTS DETECTED!!!: %s", faultId)
					}
					if len(faultIds) > 0 {
						log.Errorf("!!!RECOVER THE STORAGE FAULTS ASAP, OR YOU MIGHT LOSS THE PLEDGED ASSETS!!!")
					}
				}

				log.Infof("Reported node status[%b] to SAO network, txHash=%s", status, txHash)
			case <-ctx.Done():
				return
			}
		}
	}()
}
