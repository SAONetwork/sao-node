package client

import (
	"context"
	"crypto/rand"
	"io"

	logging "github.com/ipfs/go-log/v2"
	ic "github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	libp2pquic "github.com/libp2p/go-libp2p/p2p/transport/quic"

	ma "github.com/multiformats/go-multiaddr"
)

var log = logging.Logger("transport-client")

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
