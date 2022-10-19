package transport

import (
	"github.com/libp2p/go-libp2p/core/crypto"
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

func ServeWebsocketTransportServer(address string, serverKey crypto.PrivKey) (tpt.Listener, error) {
	addr, err := ma.NewMultiaddr(address + "/ws")
	if err != nil {
		log.Error(err.Error())
		return nil, err
	}

	serverId, u := newSecureUpgrader(serverKey)
	t, err := libp2pwebsocket.New(u, network.NullResourceManager)
	if err != nil {
		log.Error(err.Error())
		return nil, err
	}

	ln, err := t.Listen(addr)
	if err != nil {
		log.Error(err.Error())
		return nil, err
	}
	log.Info("TransportServer listening on ", ln.Multiaddr(), "/p2p/", serverId)

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

	return ln, nil
}

func newSecureUpgrader(serverKey crypto.PrivKey) (peer.ID, tpt.Upgrader) {
	id, err := peer.IDFromPrivateKey(serverKey)
	if err != nil {
		log.Error(err.Error())
		return "", nil
	}

	var secMuxer csms.SSMuxer
	noiseTpt, err := noise.New(serverKey)
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
