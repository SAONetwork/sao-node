package chain

import (
	"context"
	"encoding/hex"
	"fmt"
	"github.com/cosmos/cosmos-sdk/types/query"
	"time"

	"github.com/SaoNetwork/sao-node/types"

	sdktypes "github.com/cosmos/cosmos-sdk/types"
	coretypes "github.com/tendermint/tendermint/rpc/core/types"

	"github.com/SaoNetwork/sao-did/sid"
	didtypes "github.com/SaoNetwork/sao/x/did/types"
	modeltypes "github.com/SaoNetwork/sao/x/model/types"
	nodetypes "github.com/SaoNetwork/sao/x/node/types"
	ordertypes "github.com/SaoNetwork/sao/x/order/types"
	saotypes "github.com/SaoNetwork/sao/x/sao/types"
	"github.com/cosmos/cosmos-sdk/client"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/ignite/cli/ignite/pkg/cosmosclient"
	"github.com/ipfs/go-cid"
	logging "github.com/ipfs/go-log/v2"
	"github.com/tendermint/tendermint/rpc/client/http"
)

var log = logging.Logger("chain")

const ADDRESS_PREFIX = "sao"
const CURRENT_NET_VERSION = "v1.5.0"
const DOWNLOAD_URL = "https://github.com/SAONetwork/sao-node/releases"

// chain service provides access to cosmos chain, mainly including tx broadcast, data query, event listen.
type ChainSvc struct {
	ctx              context.Context
	cosmos           cosmosclient.Client
	bankClient       banktypes.QueryClient
	orderClient      ordertypes.QueryClient
	nodeClient       nodetypes.QueryClient
	didClient        didtypes.QueryClient
	modelClient      modeltypes.QueryClient
	listener         *http.HTTP
	accountRetriever authtypes.AccountRetriever
	ap               *AddressPool
	broadcastChanMap map[string]chan BroadcastTxJob
	stopChan         chan struct{}
}

type BroadcastTxJob struct {
	signer     string
	msg        sdktypes.Msg
	resultChan chan BroadcastTxJobResult
}

type BroadcastTxJobResult struct {
	resp cosmosclient.Response
	err  error
}

type ChainSvcApi interface {
	Stop(ctx context.Context) error
	GetLastHeight(ctx context.Context) (int64, error)
	GetAccount(ctx context.Context, address string) (client.Account, error)
	GetBalance(ctx context.Context, address string) (sdktypes.Coins, error)
	GetDidInfo(ctx context.Context, did string) (types.DidInfo, error)
	GetFishmen(ctx context.Context) (string, error)
	GetSidDocument(ctx context.Context, versionId string) (*sid.SidDocument, error)
	UpdateDidBinding(ctx context.Context, creator string, did string, accountId string) (string, error)
	QueryPaymentAddress(ctx context.Context, did string) (string, error)
	QueryMetadata(ctx context.Context, req *types.MetadataProposal, height int64) (*saotypes.QueryMetadataResponse, error)
	GetMeta(ctx context.Context, dataId string) (*modeltypes.QueryGetMetadataResponse, error)
	GetModel(ctx context.Context, key string) (*modeltypes.QueryGetModelResponse, error)
	UpdatePermission(ctx context.Context, signer string, proposal *types.PermissionProposal) (string, error)
	Create(ctx context.Context, creator string) (string, error)
	Reset(ctx context.Context, creator string, peerInfo string, status uint32, txAddresses []string, description *nodetypes.Description) (string, error)
	GetNodePeer(ctx context.Context, creator string) (string, error)
	GetNodeStatus(ctx context.Context, creator string) (uint32, error)
	ListNodes(ctx context.Context) ([]nodetypes.Node, error)
	StartStatusReporter(ctx context.Context, creator string, status uint32)
	OrderReady(ctx context.Context, provider string, orderId uint64) (saotypes.MsgReadyResponse, string, int64, error)
	StoreOrder(ctx context.Context, signer string, clientProposal *types.OrderStoreProposal) (saotypes.MsgStoreResponse, string, int64, error)
	CompleteOrder(ctx context.Context, creator string, orderId uint64, cid cid.Cid, size uint64) (string, int64, error)
	RenewOrder(ctx context.Context, creator string, orderRenewProposal types.OrderRenewProposal) (string, map[string]string, error)
	MigrateOrder(ctx context.Context, creator string, dataIds []string) (string, map[string]string, int64, error)
	GetOrder(ctx context.Context, orderId uint64) (*ordertypes.FullOrder, error)
	GetShard(ctx context.Context, shardId uint64) (*ordertypes.Shard, error)
	//SubscribeOrderComplete(ctx context.Context, orderId uint64, doneChan chan OrderCompleteResult) error
	//UnsubscribeOrderComplete(ctx context.Context, orderId uint64) error
	//SubscribeShardTask(ctx context.Context, nodeAddr string, shardTaskChan chan *ShardTask) error
	//UnsubscribeShardTask(ctx context.Context, nodeAddr string) error
	TerminateOrder(ctx context.Context, creator string, terminateProposal types.OrderTerminateProposal) (string, error)
	GetTx(ctx context.Context, hash string, heigth int64) (*coretypes.ResultTx, error)
	ReportFaults(ctx context.Context, creator string, provider string, faults []*saotypes.Fault) ([]string, error)
	RecoverFaults(ctx context.Context, creator string, provider string, faults []*saotypes.Fault) ([]string, error)
}

func NewChainSvc(
	ctx context.Context,
	chainAddress string,
	wsEndpoint string,
	keyringHome string,
) (*ChainSvc, error) {
	log.Debugf("initialize chain client")

	cosmos, err := cosmosclient.New(ctx,
		cosmosclient.WithAddressPrefix(ADDRESS_PREFIX),
		cosmosclient.WithNodeAddress(chainAddress),
		cosmosclient.WithKeyringDir(keyringHome),
		cosmosclient.WithGas("auto"),
	)
	if err != nil {
		return nil, types.Wrap(types.ErrCreateChainServiceFailed, err)
	}

	saoClient := saotypes.NewQueryClient(cosmos.Context())
	resp, err := saoClient.NetVersion(ctx, &saotypes.QueryNetVersionRequest{})
	if err != nil {
		return nil, types.Wrap(types.ErrCreateChainServiceFailed, err)
	}
	if CURRENT_NET_VERSION != resp.Version {
		return nil, fmt.Errorf("invalid net version, saonode has to be upgrade to adapt to the net verion %s. Download the the lastest saonode binary at %s", resp.Version, DOWNLOAD_URL)
	}

	accountRetriever := authtypes.AccountRetriever{}
	bankClient := banktypes.NewQueryClient(cosmos.Context())
	orderClient := ordertypes.NewQueryClient(cosmos.Context())
	nodeClient := nodetypes.NewQueryClient(cosmos.Context())
	didClient := didtypes.NewQueryClient(cosmos.Context())
	modelClient := modeltypes.NewQueryClient(cosmos.Context())

	log.Debugf("initialize chain listener")
	http, err := http.New(chainAddress, wsEndpoint)
	if err != nil {
		return nil, types.Wrap(types.ErrCreateChainServiceFailed, err)
	}

	return &ChainSvc{
		ctx:              ctx,
		cosmos:           cosmos,
		bankClient:       bankClient,
		orderClient:      orderClient,
		nodeClient:       nodeClient,
		didClient:        didClient,
		modelClient:      modelClient,
		listener:         http,
		accountRetriever: accountRetriever,
		broadcastChanMap: make(map[string]chan BroadcastTxJob),
		stopChan:         make(chan struct{}),
	}, nil
}

/**
 * add tx msg to wait channel for broadcasting.
 *
 * @param respChan is to notify broadcast result
 */
func (c *ChainSvc) broadcastMsg(signer string, msg sdktypes.Msg, respChan chan BroadcastTxJobResult) {
	if _, exists := c.broadcastChanMap[signer]; !exists {
		log.Debugf("broadcast chan for signer %s doesn't exist, create.", signer)
		c.broadcastChanMap[signer] = make(chan BroadcastTxJob, 1)
		go c.broadcastLoop(c.ctx, c.broadcastChanMap[signer])
		time.Sleep(time.Second)
	}

	c.broadcastChanMap[signer] <- BroadcastTxJob{
		signer:     signer,
		msg:        msg,
		resultChan: respChan,
	}
}

/**
 * loop for tx msg to proceed for a certain signer.
 * TODO: better to have a mechanism if a signer chan empty too long, then release this chan.
 */
func (c *ChainSvc) broadcastLoop(ctx context.Context, ch chan BroadcastTxJob) {
	log.Info("start tx broadcast loop...")
	for {
		select {
		case job := <-ch:
			signerAcc, err := c.cosmos.Account(job.signer)
			if err != nil {
				job.resultChan <- BroadcastTxJobResult{
					err: types.Wrap(types.ErrAccountNotFound, err),
				}
			} else {
				txResp, err := c.cosmos.BroadcastTx(ctx, signerAcc, job.msg)
				if err != nil {
					job.resultChan <- BroadcastTxJobResult{
						err: types.Wrap(types.ErrTxProcessFailed, err),
					}
				} else {
					job.resultChan <- BroadcastTxJobResult{
						resp: txResp,
					}
				}
			}
		case <-c.stopChan:
			log.Info("tx broadcast loop stopped.")
			return
		case <-ctx.Done():
			return
		}
	}
}

func (c *ChainSvc) SetAddressPool(ctx context.Context, ap *AddressPool) {
	c.ap = ap
}

func (c *ChainSvc) Stop(ctx context.Context) error {
	if c.listener != nil {
		log.Infof("Stop chain listener.")
		err := c.listener.Stop()
		if err != nil {
			return types.Wrap(types.ErrStopChainServiceFailed, err)
		}
	}
	c.stopChan <- struct{}{}
	for _, ch := range c.broadcastChanMap {
		close(ch)
	}
	return nil
}

func (c *ChainSvc) GetLastHeight(ctx context.Context) (int64, error) {
	return c.cosmos.LatestBlockHeight(ctx)
}

func (c *ChainSvc) GetAccount(ctx context.Context, address string) (client.Account, error) {
	accAddress, err := sdktypes.AccAddressFromBech32(address)
	if err != nil {
		return nil, types.Wrap(types.ErrSignedFailed, err)
	}

	return c.accountRetriever.GetAccount(c.cosmos.Context(), accAddress)
}

func (c *ChainSvc) GetBalance(ctx context.Context, address string) (sdktypes.Coins, error) {
	return c.cosmos.BankBalances(ctx, address, nil)
}

func(c *ChainSvc) GetBlock(ctx context.Context, height int64) (*coretypes.ResultBlock, error) {
	return c.cosmos.RPC.Block(ctx, &height)
}

func (c *ChainSvc) GetTx(ctx context.Context, hash string, height int64) (*coretypes.ResultTx, error) {
	for {
		curHeight, err := c.GetLastHeight(ctx)
		if err != nil {
			return nil, types.Wrap(types.ErrTxQueryFailed, err)
		}
		if curHeight > height {
			break
		}
		time.Sleep(time.Second)
	}
	hashBytes, err := hex.DecodeString(hash)
	if err != nil {
		return nil, types.Wrap(types.ErrTxQueryFailed, err)
	}
	return c.cosmos.RPC.Tx(ctx, hashBytes, true)
}

func (c *ChainSvc) GetFishmen(ctx context.Context) (string, error) {
	resp, err := c.nodeClient.Fishmen(ctx, &nodetypes.QueryFishmenRequest{})
	if err != nil {
		return "", types.Wrap(types.ErrCreateChainServiceFailed, err)
	}

	return resp.Fishmen, nil
}

func (c *ChainSvc) GetAllOrders(ctx context.Context, offset uint64, limit uint64) ([]ordertypes.Order, uint64, error) {
	resp, err := c.orderClient.OrderAll(ctx, &ordertypes.QueryAllOrderRequest{
		Pagination: &query.PageRequest{
			Offset: offset,
			Limit:  limit,
			Reverse: false,
			CountTotal: true,
		},
	})
	if err != nil {
		return nil, 0, types.Wrap(types.ErrQueryOrderFailed, err)
	}
	return resp.Order, resp.Pagination.Total, nil
}
