package transport

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"time"

	"github.com/SaoNetwork/sao-node/node/config"
	"github.com/SaoNetwork/sao-node/types"

	"github.com/libp2p/go-libp2p"
	libp2pwebtransport "github.com/libp2p/go-libp2p/p2p/transport/webtransport"
	ma "github.com/multiformats/go-multiaddr"

	"github.com/ipfs/go-datastore"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/network"
)

type Libp2pRpcServer struct {
	RH *RpcHandler
}

func StartLibp2pRpcServer(ctx context.Context, address string, serverKey crypto.PrivKey, db datastore.Batching, cfg *config.Node, rh *RpcHandler) (*Libp2pRpcServer, error) {
	if !cfg.Libp2p.ExternalIpEnable && !cfg.Libp2p.IntranetIpEnable && len(cfg.Libp2p.AnnounceAddresses) == 0 {
		cfg.Libp2p.ExternalIpEnable = true
		log.Warn("Intranet ip and external ip are both disabled, enable external ip as default")
	}

	tr, err := libp2pwebtransport.New(serverKey, nil, network.NullResourceManager)
	if err != nil {
		return nil, err
	}

	h, err := libp2p.New(libp2p.Transport(tr), libp2p.Identity(serverKey))
	if err != nil {
		return nil, err
	}

	err = h.Network().Listen(ma.StringCast(address + "/quic/webtransport"))
	if err != nil {
		return nil, err
	}

	var peerInfos []string
	var addressPattern string
	for _, a := range h.Addrs() {
		withP2p := a.Encapsulate(ma.StringCast("/p2p/" + h.ID().String()))
		if cfg.Libp2p.IntranetIpEnable {
			log.Debug("addr=", withP2p.String())
			peerInfos = append(peerInfos, withP2p.String())
		}
		if cfg.Libp2p.ExternalIpEnable && cfg.Libp2p.PublicAddress != "" && strings.Contains(withP2p.String(), "127.0.0.1") {
			publicAddrWithP2p := strings.Replace(withP2p.String(), "127.0.0.1", cfg.Libp2p.PublicAddress, 1)
			log.Debug("addr=", publicAddrWithP2p)
			peerInfos = append(peerInfos, publicAddrWithP2p)
		}
		if strings.Contains(a.String(), "/ip4/127.0.0.1/udp/5154") {
			addressPattern = a.Encapsulate(ma.StringCast("/p2p/" + h.ID().String())).String()
		}
	}
	if len(cfg.Libp2p.AnnounceAddresses) > 0 {
		announceAddresses := make([]string, 0)
		for _, address := range cfg.Libp2p.AnnounceAddresses {
			if strings.Contains(address, "udp") {
				announceAddresses = append(announceAddresses, strings.ReplaceAll(addressPattern, "/ip4/127.0.0.1/udp/5154", address))
			}
		}
		if len(announceAddresses) > 0 {
			peerInfos = append(peerInfos, strings.Join(announceAddresses, ","))
		}
	}

	if len(peerInfos) > 0 {
		key := datastore.NewKey(fmt.Sprintf(types.PEER_INFO_PREFIX))
		peers, err := db.Get(ctx, key)
		if err != nil {
			return nil, err
		}
		if len(peers) > 0 {
			db.Put(ctx, key, []byte(string(peers)+","+strings.Join(peerInfos, ",")))
		} else {
			db.Put(ctx, key, []byte(strings.Join(peerInfos, ",")))
		}
	}

	rs := &Libp2pRpcServer{
		RH: rh,
	}

	h.Network().SetStreamHandler(rs.HandleStream)

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	select {
	case <-c:
	case <-time.After(time.Second):
	}

	return rs, nil
}

func (rs *Libp2pRpcServer) HandleStream(s network.Stream) {
	defer s.Close()

	// Set a deadline on reading from the stream so it doesn’t hang
	_ = s.SetReadDeadline(time.Now().Add(30 * time.Second))
	defer s.SetReadDeadline(time.Time{}) // nolint

	var req types.RpcReq
	var resp = types.RpcResp{}

	buf := &bytes.Buffer{}
	buf.ReadFrom(s)
	err := json.Unmarshal(buf.Bytes(), &req)
	if err == nil {
		log.Info("Got rpc request: ", req.Method)

		var result string
		var err error
		switch req.Method {
		case "Sao.Upload":
			req.Params = append(req.Params, filepath.Join(rs.RH.StagingPath, s.Conn().RemotePeer().String()))
			result, err = rs.RH.Upload(req.Params)
		case "Sao.ModelCreate":
			result, err = rs.RH.Create(req.Params)
		case "Sao.ModelLoad":
			result, err = rs.RH.Load(req.Params)
		case "Sao.ModelUpdate":
			result, err = rs.RH.Update(req.Params)
		case "Did.BindingProof":
			result, err = rs.RH.BindingProof(req.Params)
		default:
			resp.Error = "N/a"
		}
		if err != nil {
			resp.Error = err.Error()
		} else {
			resp.Data = result
		}

	} else {
		resp.Error = err.Error()
	}

	bytes, err := json.Marshal(resp)
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

	log.Info("Sent rpc response: ", resp)
}
