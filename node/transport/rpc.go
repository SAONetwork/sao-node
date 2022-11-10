package transport

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sao-storage-node/node/config"
	"sao-storage-node/types"
	"strings"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p"
	libp2pwebsocket "github.com/libp2p/go-libp2p/p2p/transport/websocket"
	ma "github.com/multiformats/go-multiaddr"

	"github.com/ipfs/go-datastore"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/p2p/muxer/yamux"
	csms "github.com/libp2p/go-libp2p/p2p/net/conn-security-multistream"
	tptu "github.com/libp2p/go-libp2p/p2p/net/upgrader"
	noise "github.com/libp2p/go-libp2p/p2p/security/noise"
)

type RpcServer struct {
	Ctx  context.Context
	DbLk sync.Mutex
	Db   datastore.Batching
}

func StartRpcServer(ctx context.Context, address string, serverKey crypto.PrivKey, db datastore.Batching, cfg *config.Node) (*RpcServer, error) {
	var secMuxer csms.SSMuxer
	noiseTpt, err := noise.New(serverKey)
	if err != nil {
		return nil, err
	}
	secMuxer.AddTransport(noise.ID, noiseTpt)

	u, err := tptu.New(&secMuxer, yamux.DefaultTransport)
	if err != nil {
		return nil, err
	}

	tr, err := libp2pwebsocket.New(u, network.NullResourceManager)
	if err != nil {
		return nil, err
	}

	h, err := libp2p.New(libp2p.Transport(tr), libp2p.Identity(serverKey))
	if err != nil {
		log.Error(err.Error())
		return nil, err
	}

	err = h.Network().Listen(ma.StringCast(address + "/ws"))
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
		Ctx: ctx,
		Db:  db,
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

	log.Info("lao6.............")

	// Set a deadline on reading from the stream so it doesn’t hang
	_ = s.SetReadDeadline(time.Now().Add(30 * time.Second))
	defer s.SetReadDeadline(time.Time{}) // nolint

	if err := s.CloseWrite(); err != nil {
		log.Error(err.Error())
		return
	}
}
