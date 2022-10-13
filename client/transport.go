package client

import (
	"context"
	"crypto/rand"
	"io"

	cid "github.com/ipfs/go-cid"
	logging "github.com/ipfs/go-log/v2"
	ic "github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	libp2pquic "github.com/libp2p/go-libp2p/p2p/transport/quic"
	mc "github.com/multiformats/go-multicodec"
	mh "github.com/multiformats/go-multihash"

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

	pref := cid.Prefix{
		Version:  1,
		Codec:    uint64(mc.Raw),
		MhType:   mh.SHA2_256,
		MhLength: -1, // default length
	}
	cidLocal, err := pref.Sum(content)
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
	res, err := io.ReadAll(str)
	if err != nil {
		log.Error(err)
		return nil
	}
	log.Debug("Received ", len(res), " bytes...")
	_, cidRemote, err := cid.CidFromBytes(res)
	if err != nil {
		log.Error(err)
		return nil
	}

	if cidRemote.Equals(cidLocal) {
		return res
	} else {
		log.Error("file cid mismatch, ", cidLocal.String(), " vs. ", cidRemote.String())
		return nil
	}
}
