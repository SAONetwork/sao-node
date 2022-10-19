package transport

import (
	"io"

	cid "github.com/ipfs/go-cid"
	logging "github.com/ipfs/go-log/v2"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	tpt "github.com/libp2p/go-libp2p/core/transport"
	libp2pwebtransport "github.com/libp2p/go-libp2p/p2p/transport/webtransport"
	ma "github.com/multiformats/go-multiaddr"
	mc "github.com/multiformats/go-multicodec"
	mh "github.com/multiformats/go-multihash"
)

var log = logging.Logger("transport")

func StartWebTransportServer(address string, serverKey crypto.PrivKey) (tpt.Listener, error) {
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

	serverId, err := peer.IDFromPrivateKey(serverKey)
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
