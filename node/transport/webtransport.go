package transport

import (
	"fmt"
	"sao-storage-node/node/repo"

	logging "github.com/ipfs/go-log/v2"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/transport"
	libp2pwebtransport "github.com/libp2p/go-libp2p/p2p/transport/webtransport"
	ma "github.com/multiformats/go-multiaddr"
)

var log = logging.Logger("transport")

func StartWebTransportServer(address string, repo *repo.Repo) (transport.Listener, error) {
	serverKey, err := repo.PeerId()
	if err != nil {
		log.Error(err.Error())
		return nil, err
	}

	peerID, err := peer.IDFromPrivateKey(serverKey)
	if err != nil {
		log.Error(err.Error())
		return nil, err
	}

	tr, err := libp2pwebtransport.New(serverKey, nil, network.NullResourceManager)
	if err != nil {
		log.Error(err.Error())
		return nil, err
	}
	ln, err := tr.Listen(ma.StringCast(address + "/quic/webtransport"))
	if err != nil {
		log.Error(err.Error())
		return nil, err
	}
	fmt.Println("Listening on ", peerID, " (", ln.Multiaddr(), ")")

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

	return ln, nil
}
