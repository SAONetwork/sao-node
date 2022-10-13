package transport

import (
	"sao-storage-node/node/repo"

	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	tpt "github.com/libp2p/go-libp2p/core/transport"
	"github.com/libp2p/go-libp2p/p2p/muxer/yamux"
	csms "github.com/libp2p/go-libp2p/p2p/net/conn-security-multistream"
	tptu "github.com/libp2p/go-libp2p/p2p/net/upgrader"
	noise "github.com/libp2p/go-libp2p/p2p/security/noise"
	libp2pwebsocket "github.com/libp2p/go-libp2p/p2p/transport/websocket"
	ma "github.com/multiformats/go-multiaddr"
)

func ServeWebsocketTransport(address string, repo *repo.Repo) error {
	addr, err := ma.NewMultiaddr(address + "/ws/webtransport")
	if err != nil {
		log.Error(err.Error())
		return err
	}

	peerID, u := newSecureUpgrader(repo)
	t, err := libp2pwebsocket.New(u, network.NullResourceManager)
	if err != nil {
		log.Error(err.Error())
		return err
	}

	ln, err := t.Listen(addr)
	if err != nil {
		log.Error(err.Error())
		return err
	}
	log.Info("Listening on ", peerID, " (", ln.Multiaddr(), ")")

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				log.Error(err.Error())
				continue
			}
			log.Info("Accepted new connection from ", conn.RemotePeer(), " (", conn.RemoteMultiaddr(), ")")
			go func() {
				if err := handleConn(conn); err != nil {
					log.Error("handling conn failed: ", err.Error())
				}
			}()
		}
	}()

	return err
}

func newSecureUpgrader(repo *repo.Repo) (peer.ID, tpt.Upgrader) {
	priv, err := repo.PeerId()
	if err != nil {
		log.Error(err.Error())
		return "", nil
	}
	id, err := peer.IDFromPrivateKey(priv)
	if err != nil {
		log.Error(err.Error())
		return "", nil
	}

	var secMuxer csms.SSMuxer
	noiseTpt, err := noise.New(priv)
	if err != nil {
		log.Error(err.Error())
		return "", nil
	}
	secMuxer.AddTransport(noise.ID, noiseTpt)

	u, err := tptu.New(&secMuxer, yamux.DefaultTransport)
	if err != nil {
		log.Error(err.Error())
	}
	return id, u
}
