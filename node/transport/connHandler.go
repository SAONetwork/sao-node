package transport

import (
	"io"

	cid "github.com/ipfs/go-cid"
	tpt "github.com/libp2p/go-libp2p/core/transport"
	mc "github.com/multiformats/go-multicodec"
	mh "github.com/multiformats/go-multihash"
)

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
