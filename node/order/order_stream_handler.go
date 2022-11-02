package order

import (
	"sao-storage-node/types"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p/core/network"
)

type ShardStreamHandler struct {
	stagingPath string
}

var (
	handler *ShardStreamHandler
	once    sync.Once
)

func NewShardStreamHandler(path string) *ShardStreamHandler {
	once.Do(func() {
		handler = &ShardStreamHandler{
			stagingPath: path,
		}
	})

	return handler
}

func (ssh *ShardStreamHandler) HandleShardStream(s network.Stream) {
	defer s.Close()

	// Set a deadline on reading from the stream so it doesn't hang
	_ = s.SetReadDeadline(time.Now().Add(10 * time.Second))
	defer s.SetReadDeadline(time.Time{}) // nolint

	var req types.ShardStoreReq
	err := req.Unmarshal(s, "json")
	if err != nil {
		log.Error(err)
		// TODO: respond error
		return
	}
	log.Debugf("receive ShardStoreReq: orderId=%d cid=%v", req.OrderId, req.Cid)

	contentBytes, err := GetStagedShard(ssh.stagingPath, req.Owner, req.Cid)
	if err != nil {
		log.Error(err)
		// TODO: respond error
		return
	}
	var resp = &types.ShardStoreResp{
		OrderId: req.OrderId,
		Cid:     req.Cid,
		Content: contentBytes,
	}
	log.Debugf("send ShardStoreResp: Content=%v", string(contentBytes))

	err = resp.Marshal(s, "json")
	if err != nil {
		log.Error(err.Error())
		return
	}

	if err := s.CloseWrite(); err != nil {
		log.Error(err.Error())
		return
	}
}
