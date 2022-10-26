package client

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"io"
	"os"

	cid "github.com/ipfs/go-cid"
	logging "github.com/ipfs/go-log/v2"
	"github.com/libp2p/go-libp2p/core/network"

	ic "github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	libp2pwebtransport "github.com/libp2p/go-libp2p/p2p/transport/webtransport"

	mc "github.com/multiformats/go-multicodec"
	mh "github.com/multiformats/go-multihash"

	ma "github.com/multiformats/go-multiaddr"
)

var log = logging.Logger("transport-client")

const CHUNK_SIZE int = 32 * 1024 * 1024

type FileChunkReq struct {
	ChunkId     int
	TotalLength int
	TotalChunks int
	ChunkCid    string
	Cid         string
	Content     []byte
}

func DoWebTransport(ctx context.Context, remoteAddr string, remotePeerId string, fpath string) cid.Cid {
	file, err := os.Open(fpath)
	if err != nil {
		log.Error(err)
		return cid.Undef
	}

	serverAddress, err := ma.NewMultiaddr(remoteAddr)
	if err != nil {
		log.Error(err)
		return cid.Undef
	}

	serverId, err := peer.Decode(remotePeerId)
	if err != nil {
		log.Error(err)
		return cid.Undef
	}

	clientKey, _, err := ic.GenerateEd25519Key(rand.Reader)
	if err != nil {
		log.Error(err)
		return cid.Undef
	}

	tr, err := libp2pwebtransport.New(clientKey, nil, network.NullResourceManager)
	if err != nil {
		log.Error(err)
		return cid.Undef
	}

	log.Info("Dialing ", serverId, " (", serverAddress, ")")
	conn, err := tr.Dial(ctx, serverAddress, serverId)
	if err != nil {
		log.Error(err)
		return cid.Undef
	}
	defer conn.Close()

	content, err := io.ReadAll(file)
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
	contentCid, err := pref.Sum(content)
	if err != nil {
		log.Error(err)
		return cid.Undef
	}

	var contentLength int = len(content)
	var totalChunks = contentLength/CHUNK_SIZE + 1
	chunkId := 0
	for chunkId <= totalChunks {
		var chunk []byte
		if (chunkId+1)*CHUNK_SIZE < len(content) {
			chunk = content[chunkId*CHUNK_SIZE : (chunkId+1)*CHUNK_SIZE]
		} else if chunkId*CHUNK_SIZE < len(content) {
			chunk = content[chunkId*CHUNK_SIZE:]
		} else {
			chunk = make([]byte, 0)
		}

		pref := cid.Prefix{
			Version:  1,
			Codec:    uint64(mc.Raw),
			MhType:   mh.SHA2_256,
			MhLength: -1, // default length
		}
		chunkCid, err := pref.Sum(chunk)
		if err != nil {
			log.Error(err)
			return cid.Undef
		}

		log.Info("Content[", chunkId, "], CID: ", chunkCid, ", length: ", len(chunk))

		str, err := conn.OpenStream(ctx)
		if err != nil {
			log.Error(err)
			return cid.Undef
		}
		defer str.Close()

		req := &FileChunkReq{
			ChunkId:     chunkId,
			TotalLength: contentLength,
			TotalChunks: totalChunks,
			ChunkCid:    chunkCid.String(),
			Cid:         contentCid.String(),
			Content:     chunk,
		}
		bytes, err := json.Marshal(req)
		if err != nil {
			log.Error(err)
			return cid.Undef
		}

		if _, err := str.Write(bytes); err != nil {
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
		remoteCid := string(res)

		if remoteCid == chunkCid.String() {
			chunkId++
		} else if remoteCid == contentCid.String() && len(chunk) == 0 {
			break
		} else {
			log.Error("file cid mismatch, ", chunkCid, " vs. ", remoteCid)
			return cid.Undef
		}
	}

	return contentCid
}
