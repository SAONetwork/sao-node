package client

import (
	"context"
	"crypto/rand"
	"io"

	cid "github.com/ipfs/go-cid"
	logging "github.com/ipfs/go-log/v2"
	"github.com/libp2p/go-libp2p/core/network"
	csms "github.com/libp2p/go-libp2p/p2p/net/conn-security-multistream"
	tptu "github.com/libp2p/go-libp2p/p2p/net/upgrader"

	ic "github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/muxer/yamux"
	noise "github.com/libp2p/go-libp2p/p2p/security/noise"
	libp2pquic "github.com/libp2p/go-libp2p/p2p/transport/quic"
	libp2pwebsocket "github.com/libp2p/go-libp2p/p2p/transport/websocket"
	mc "github.com/multiformats/go-multicodec"
	mh "github.com/multiformats/go-multihash"

	ma "github.com/multiformats/go-multiaddr"
)

var log = logging.Logger("transport-client")

func DoWebsocketTransport(ctx context.Context, remoteAddr string, remotePeerId string, content []byte) cid.Cid {
	address, err := ma.NewMultiaddr(remoteAddr)
	if err != nil {
		log.Error(err)
		return cid.Undef
	}
	peerId, err := peer.Decode(remotePeerId)
	if err != nil {
		log.Error(err)
		return cid.Undef
	}

	priv, _, err := ic.GenerateECDSAKeyPair(rand.Reader)
	if err != nil {
		log.Error(err)
		return cid.Undef
	}

	var secMuxer csms.SSMuxer
	noiseTpt, err := noise.New(priv)
	if err != nil {
		log.Error(err.Error())
		return cid.Undef
	}
	secMuxer.AddTransport(noise.ID, noiseTpt)

	u, err := tptu.New(&secMuxer, yamux.DefaultTransport)
	if err != nil {
		log.Error(err.Error())
	}

	t, err := libp2pwebsocket.New(u, network.NullResourceManager)
	if err != nil {
		log.Error(err)
		return cid.Undef
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
		return cid.Undef
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
		return cid.Undef
	}
	defer str.Close()

	log.Debug("Sending ", len(content), " bytes...")
	if _, err := str.Write(content); err != nil {
		log.Error(err)
		return cid.Undef
	}
	if err := str.CloseWrite(); err != nil {
		log.Error(err)
		return cid.Undef
	}
	res, err := io.ReadAll(str)
	if err != nil {
		log.Error(err)
		return cid.Undef
	}
	log.Debug("Received ", len(res), " bytes...")
	_, cidRemote, err := cid.CidFromBytes(res)
	if err != nil {
		log.Error(err)
		return cid.Undef
	}

	if cidRemote.Equals(cidLocal) {
		return cidRemote
	} else {
		log.Error("file cid mismatch, ", cidLocal.String(), " vs. ", cidRemote.String())
		return cid.Undef
	}
}

func DoQuicTransport(ctx context.Context, remoteAddr string, remotePeerId string, content []byte) cid.Cid {
	address, err := ma.NewMultiaddr(remoteAddr)
	if err != nil {
		log.Error(err)
		return cid.Undef
	}
	peerId, err := peer.Decode(remotePeerId)
	if err != nil {
		log.Error(err)
		return cid.Undef
	}

	priv, _, err := ic.GenerateECDSAKeyPair(rand.Reader)
	if err != nil {
		log.Error(err)
		return cid.Undef
	}

	t, err := libp2pquic.NewTransport(priv, nil, nil, nil)
	if err != nil {
		log.Error(err)
		return cid.Undef
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
		return cid.Undef
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
		return cid.Undef
	}
	defer str.Close()

	log.Debug("Sending ", len(content), " bytes...")
	if _, err := str.Write(content); err != nil {
		log.Error(err)
		return cid.Undef
	}
	if err := str.CloseWrite(); err != nil {
		log.Error(err)
		return cid.Undef
	}
	res, err := io.ReadAll(str)
	if err != nil {
		log.Error(err)
		return cid.Undef
	}
	log.Debug("Received ", len(res), " bytes...")
	_, cidRemote, err := cid.CidFromBytes(res)
	if err != nil {
		log.Error(err)
		return cid.Undef
	}

	if cidRemote.Equals(cidLocal) {
		return cidRemote
	} else {
		log.Error("file cid mismatch, ", cidLocal.String(), " vs. ", cidRemote.String())
		return cid.Undef
	}
}
