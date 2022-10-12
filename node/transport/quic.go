package transport

import (
	"context"
	"crypto/rand"
	"io"

	"sao-storage-node/node/repo"

	logging "github.com/ipfs/go-log/v2"
	ic "github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	tpt "github.com/libp2p/go-libp2p/core/transport"
	libp2pquic "github.com/libp2p/go-libp2p/p2p/transport/quic"

	ma "github.com/multiformats/go-multiaddr"
)

var log = logging.Logger("transport")

func ServeQuicTransport(address string, repo *repo.Repo) error {
	addr, err := ma.NewMultiaddr(address + "/quic")

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

func handleConn(conn tpt.CapableConn) error {
	str, err := conn.AcceptStream()
	if err != nil {
		return err
	}
	data, err := io.ReadAll(str)
	if err != nil {
		return err
	}
	log.Debug("Received ", len(data), " bytes...")
	if _, err := str.Write([]byte("OK")); err != nil {
		return err
	}
	return str.Close()
}

func DoQuicTransport(ctx context.Context, remoteAddr string, remotePeerId string, content []byte) []byte {
	address, err := ma.NewMultiaddr(remoteAddr)
	if err != nil {
		log.Error(err)
		return nil
	}
	peerId, err := peer.Decode(remotePeerId)
	if err != nil {
		log.Error(err)
		return nil
	}

	priv, _, err := ic.GenerateECDSAKeyPair(rand.Reader)
	if err != nil {
		log.Error(err)
		return nil
	}

	t, err := libp2pquic.NewTransport(priv, nil, nil, nil)
	if err != nil {
		log.Error(err)
		return nil
	}

	log.Info("Dialing ", remoteAddr, " (", remotePeerId, ")")
	conn, err := t.Dial(context.Background(), address, peerId)
	if err != nil {
		log.Error(err)
	}
	defer conn.Close()
	str, err := conn.OpenStream(context.Background())
	if err != nil {
		log.Error(err)
		return nil
	}
	defer str.Close()

	log.Debug("Sending ", len(content), " bytes...")
	if _, err := str.Write(content); err != nil {
		log.Error(err)
		return nil
	}
	if err := str.CloseWrite(); err != nil {
		log.Error(err)
		return nil
	}
	data, err := io.ReadAll(str)
	if err != nil {
		log.Error(err)
		return nil
	}
	log.Debug("Received ", len(data), " bytes...")

	return data
}
