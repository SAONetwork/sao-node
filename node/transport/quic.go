package transport

import (
	"context"
	"crypto/rand"
	"io"

	"sao-storage-node/node/repo"

	cid "github.com/ipfs/go-cid"
	mc "github.com/multiformats/go-multicodec"
	mh "github.com/multiformats/go-multihash"

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
		log.Error(err.Error())
		return err
	}
	data, err := io.ReadAll(str)
	if err != nil {
		log.Error(err.Error())
		return err
	}
	log.Debug("Received ", len(data), " bytes...")

	pref := cid.Prefix{
		Version:  1,
		Codec:    uint64(mc.Raw),
		MhType:   mh.SHA2_256,
		MhLength: -1, // default length
	}
	cid, err := pref.Sum(data)
	if err != nil {
		log.Error(err.Error())
		return err
	}
	log.Info("CID is ", cid.String())

	if _, err := str.Write(cid.Bytes()); err != nil {
		log.Error(err.Error())
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
