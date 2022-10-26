package transport

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/signal"
	"sync"
	"time"

	cid "github.com/ipfs/go-cid"
	"github.com/ipfs/go-datastore"
	logging "github.com/ipfs/go-log/v2"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/network"
	libp2pwebtransport "github.com/libp2p/go-libp2p/p2p/transport/webtransport"
	ma "github.com/multiformats/go-multiaddr"
	mc "github.com/multiformats/go-multicodec"
	mh "github.com/multiformats/go-multihash"
)

var log = logging.Logger("transport")

const CHUNCK_SIZE = 16 * 1024 * 1024

type ChunkUnmarshaler interface {
	UnMarshal(io.Reader) error
}

type ChunkReq struct {
	ChunkId     int
	TotalLength int
	TotalChunks int
	ChunkCid    string
	Cid         string
	Content     []byte
}
type ReceivedFileInfo struct {
	Cid            string
	TotalLength    int
	TotalChunks    int
	ReceivedLength int
	ChunkCids      []string
}

type TransportServer struct {
	Ctx  context.Context
	DbLk sync.Mutex
	Db   datastore.Batching
}

func StartTransportServer(ctx context.Context, address string, serverKey crypto.PrivKey, db datastore.Batching) (*TransportServer, error) {
	tr, err := libp2pwebtransport.New(serverKey, nil, network.NullResourceManager)
	if err != nil {
		log.Error(err.Error())
		return nil, err
	}

	h, err := libp2p.New(libp2p.Transport(tr), libp2p.Identity(serverKey))
	if err != nil {
		log.Error(err.Error())
		return nil, err
	}

	err = h.Network().Listen(ma.StringCast(address + "/quic/webtransport"))
	if err != nil {
		log.Error(err.Error())
		return nil, err
	}

	for _, a := range h.Addrs() {
		withP2p := a.Encapsulate(ma.StringCast("/p2p/" + h.ID().String()))
		log.Info("addr=", withP2p.String())
	}

	ts := &TransportServer{
		Ctx: ctx,
		Db:  db,
	}

	h.Network().SetStreamHandler(ts.HandleStream)

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	select {
	case <-c:
	case <-time.After(time.Second):
	}

	return ts, nil
}

func (ts *TransportServer) HandleStream(s network.Stream) {
	defer s.Close()

	// Set a deadline on reading from the stream so it doesn’t hang
	_ = s.SetReadDeadline(time.Now().Add(30 * time.Second))
	defer s.SetReadDeadline(time.Time{}) // nolint

	var req ChunkReq
	buf := &bytes.Buffer{}
	buf.ReadFrom(s)
	err := json.Unmarshal(buf.Bytes(), &req)
	if err != nil {
		log.Error(err.Error())
		return
	}

	pref := cid.Prefix{
		Version:  1,
		Codec:    uint64(mc.Raw),
		MhType:   mh.SHA2_256,
		MhLength: -1, // default length
	}
	cid, err := pref.Sum(req.Content)
	if err != nil {
		log.Error(err.Error())
		return
	}

	if len(req.Content) > 0 {
		ts.DbLk.Lock()
		defer ts.DbLk.Unlock()

		var fileInfo *ReceivedFileInfo
		key := datastore.NewKey(fmt.Sprintf("fileIno_%s", req.Cid))

		if req.ChunkId == 0 {
			fileInfo = &ReceivedFileInfo{
				Cid:            req.Cid,
				TotalLength:    req.TotalLength,
				TotalChunks:    req.TotalChunks,
				ReceivedLength: len(req.Content),
				ChunkCids:      make([]string, req.TotalChunks),
			}
			fileInfo.ChunkCids[0] = req.ChunkCid
		} else if info, err := ts.Db.Get(ts.Ctx, key); err == nil {
			err := json.Unmarshal(info, &fileInfo)
			if err != nil {
				log.Error(err.Error())
				return
			}

			if fileInfo.ChunkCids[req.ChunkId] == "" {
				fileInfo.ChunkCids[req.ChunkId] = req.ChunkCid
				fileInfo.ReceivedLength += len(req.Content)
			} else {
				log.Error("invalid chunk ", req.Cid, ", received already")
				return
			}
		} else {
			// should not happen
			log.Error("invalid req, ", err.Error())
			return
		}

		info, err := json.Marshal(fileInfo)
		if err != nil {
			log.Error(err.Error())
			return
		}

		err = ts.Db.Put(ts.Ctx, key, info)
		if err != nil {
			log.Error(err.Error())
			return
		}
	}

	log.Info("Received file chunk: ", req.ChunkId)
	log.Info("Received file chunk, remote CID: ", req.ChunkCid)
	log.Info("Received file chunk, local CID: ", cid)

	if len(req.Content) == 0 {
		// Transport is done
		if _, err := s.Write([]byte(req.Cid)); err != nil {
			log.Error(err.Error())
			return
		}
		log.Info("Received file, CID: ", req.Cid)
		log.Info("Received file, length: ", req.TotalLength)
	} else {
		if _, err := s.Write([]byte(cid.String())); err != nil {
			log.Error(err.Error())
			return
		}
	}

	if err := s.CloseWrite(); err != nil {
		log.Error(err.Error())
		return
	}
}
