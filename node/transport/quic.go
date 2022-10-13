package transport

import (
	"sao-storage-node/node/repo"

	logging "github.com/ipfs/go-log/v2"
	"github.com/libp2p/go-libp2p/core/peer"
	libp2pquic "github.com/libp2p/go-libp2p/p2p/transport/quic"

	ma "github.com/multiformats/go-multiaddr"
)

var log = logging.Logger("transport")

func ServeQuicTransport(address string, repo *repo.Repo) error {
	addr, err := ma.NewMultiaddr(address + "/quic/webtransport")
	if err != nil {
		log.Error(err.Error())
		return err
	}

	priv, err := repo.PeerId()
	if err != nil {
		log.Error(err.Error())
		return err
	}
	peerID, err := peer.IDFromPrivateKey(priv)
	if err != nil {
		log.Error(err.Error())
		return err
	}

	t, err := libp2pquic.NewTransport(priv, nil, nil, nil)
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

	return nil
}
