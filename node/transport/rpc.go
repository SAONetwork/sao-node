package transport

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"sao-storage-node/api"
	"sao-storage-node/node/config"
	"sao-storage-node/types"
	"strings"
	"sync"
	"time"

	apitypes "sao-storage-node/api/types"

	"github.com/libp2p/go-libp2p"
	libp2pwebtransport "github.com/libp2p/go-libp2p/p2p/transport/webtransport"
	ma "github.com/multiformats/go-multiaddr"

	"github.com/ipfs/go-datastore"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/network"
)

type RpcServer struct {
	Ctx        context.Context
	DbLk       sync.Mutex
	Db         datastore.Batching
	GatewayApi api.GatewayApi
}

func StartRpcServer(ctx context.Context, ga api.GatewayApi, address string, serverKey crypto.PrivKey, db datastore.Batching, cfg *config.Node) (*RpcServer, error) {
	tr, err := libp2pwebtransport.New(serverKey, nil, network.NullResourceManager)
	if err != nil {
		log.Error(err.Error())
		return nil, err
	}

	h, err := libp2p.New(libp2p.Transport(tr), libp2p.Identity(serverKey))
	if err != nil {
		log.Error(err.Error())
		return nil, err
	}

	err = h.Network().Listen(ma.StringCast(address + "/quic/webtransport"))
	if err != nil {
		log.Error(err.Error())
		return nil, err
	}

	var peerInfos []string
	for _, a := range h.Addrs() {
		withP2p := a.Encapsulate(ma.StringCast("/p2p/" + h.ID().String()))
		log.Info("addr=", withP2p.String())
		peerInfos = append(peerInfos, withP2p.String())
	}
	if len(peerInfos) > 0 {
		key := datastore.NewKey(fmt.Sprintf(types.PEER_INFO_PREFIX))
		db.Put(ctx, key, []byte(strings.Join(peerInfos, ", ")))
	}

	rs := &RpcServer{
		Ctx:        ctx,
		Db:         db,
		GatewayApi: ga,
	}

	h.Network().SetStreamHandler(rs.HandleStream)
	// h.SetStreamHandler(types.ShardLoadProtocol, HandleRpc)

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	select {
	case <-c:
	case <-time.After(time.Second):
	}

	return rs, nil
}

func (rs *RpcServer) HandleStream(s network.Stream) {
	defer s.Close()

	// Set a deadline on reading from the stream so it doesn’t hang
	_ = s.SetReadDeadline(time.Now().Add(30 * time.Second))
	defer s.SetReadDeadline(time.Time{}) // nolint

	var req types.RpcReq
	buf := &bytes.Buffer{}
	buf.ReadFrom(s)
	err := json.Unmarshal(buf.Bytes(), &req)
	if err != nil {
		log.Error(err.Error())
		return
	}

	var result string
	switch req.Method {
	case "Sao.Load":
		var loadReq apitypes.LoadReq
		err := json.Unmarshal([]byte(req.Params[0]), &loadReq)
		if err != nil {
			log.Error(err.Error())
			return
		}
		loadRes, err := rs.GatewayApi.Load(rs.Ctx, loadReq)
		if err != nil {
			log.Error(err.Error())
			return
		}
		b, err := json.Marshal(loadRes)
		if err != nil {
			log.Error(err.Error())
			return
		}
		result = string(b)
	default:
		result = "N/a"
	}

	var res = types.RpcRes{
		Data:  result,
		Error: "N/a",
	}
	bytes, err := json.Marshal(res)
	if err != nil {
		log.Error(err.Error())
		return
	}

	if _, err := s.Write(bytes); err != nil {
		log.Error(err.Error())
		return
	}

	if err := s.CloseWrite(); err != nil {
		log.Error(err.Error())
		return
	}

	log.Info("Got rpc request: ", req)
	log.Info("Sent rpc response: ", res)
}
