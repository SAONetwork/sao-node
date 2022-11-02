package transport

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"sao-storage-node/node/config"
	"sao-storage-node/types"
	"strings"
	"sync"
	"time"

	cid "github.com/ipfs/go-cid"
	"github.com/ipfs/go-datastore"
	logging "github.com/ipfs/go-log/v2"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/network"
	libp2pwebtransport "github.com/libp2p/go-libp2p/p2p/transport/webtransport"
	"github.com/mitchellh/go-homedir"
	ma "github.com/multiformats/go-multiaddr"
	mc "github.com/multiformats/go-multicodec"
	mh "github.com/multiformats/go-multihash"
)

var log = logging.Logger("transport")

type TransportServer struct {
	Ctx              context.Context
	DbLk             sync.Mutex
	Db               datastore.Batching
	StagingPath      string
	StagingSapceSize int64
}

func StartTransportServer(ctx context.Context, address string, serverKey crypto.PrivKey, db datastore.Batching, cfg *config.Node) (*TransportServer, error) {
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

	var peerInfos []string
	for _, a := range h.Addrs() {
		withP2p := a.Encapsulate(ma.StringCast("/p2p/" + h.ID().String()))
		log.Info("addr=", withP2p.String())
		peerInfos = append(peerInfos, withP2p.String())
	}
	if len(peerInfos) > 0 {
		key := datastore.NewKey(fmt.Sprintf(types.PEER_INFO_PREFIX))
		db.Put(ctx, key, []byte(strings.Join(peerInfos, ", ")))
	}

	path, err := homedir.Expand(cfg.Transport.StagingPath)
	if err != nil {
		return nil, err
	}

	err = os.MkdirAll(path, 0755)
	if err != nil && !os.IsExist(err) {
		log.Error(err.Error())
		return nil, err
	}

	ts := &TransportServer{
		Ctx:              ctx,
		Db:               db,
		StagingPath:      cfg.Transport.StagingPath,
		StagingSapceSize: cfg.Transport.StagingSapceSize,
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

	var req types.FileChunkReq
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
	localCid, err := pref.Sum(req.Content)
	if err != nil {
		log.Error(err.Error())
		return
	}

	if len(req.Content) > 0 {
		info, err := os.Stat(ts.StagingPath)
		if err != nil {
			log.Error(err.Error())
			return
		} else {
			if info.Size()+int64(len(req.Content)) > ts.StagingSapceSize {
				log.Errorf("not enough staging space under %s, need %v but only %v left", ts.StagingPath, len(req.Content), ts.StagingSapceSize-info.Size())
				return
			}
		}

		var path = filepath.Join(ts.StagingPath, s.Conn().RemotePeer().String(), req.Cid)

		ts.handleChunkInfo(&req, path)

		if _, err := s.Write([]byte(localCid.String())); err != nil {
			log.Error(err.Error())
			return
		}

		path, err = homedir.Expand(path)
		if err != nil {
			return
		}
		log.Info("path: ", path)
		_, err = os.Open(path)
		if err != nil {
			if !os.IsExist(err) {
				err = os.MkdirAll(path, 0755)
				if err != nil && !os.IsExist(err) {
					log.Error(err.Error())
					return
				}
			} else {
				log.Error(err)
				return
			}
		}

		file, err := os.Create(filepath.Join(path, req.ChunkCid))
		if err != nil {
			log.Error(err.Error())
			return
		}

		_, err = file.Write(req.Content)
		if err != nil {
			log.Error(err.Error())
			return
		}

		log.Info("Received file chunk[%d], remote CID: %s, local CID: %s", req.ChunkId, req.ChunkCid, localCid)
		log.Infof("Staging file %s generated", filepath.Join(path, req.ChunkCid))
	} else {
		// Transport is done
		if _, err := s.Write([]byte(req.Cid)); err != nil {
			log.Error(err.Error())
			return
		}

		key := datastore.NewKey(types.FILE_INFO_PREFIX + req.Cid)
		if info, err := ts.Db.Get(ts.Ctx, key); err == nil {
			var fileInfo *types.ReceivedFileInfo
			err := json.Unmarshal(info, &fileInfo)
			if err != nil {
				log.Error(err.Error())
				return
			}

			basePath, err := homedir.Expand(fileInfo.Path)
			if err != nil {
				return
			}
			log.Info("path: ", basePath)

			var fileContent []byte
			for _, chunkCid := range fileInfo.ChunkCids {
				var path = filepath.Join(basePath, chunkCid)
				file, err := os.Open(path)
				if err != nil {
					log.Error(err)
					return
				}

				content, err := io.ReadAll(file)
				if err != nil {
					log.Error(err)
					return
				}

				fileContent = append(fileContent, content...)
			}

			pref := cid.Prefix{
				Version:  1,
				Codec:    uint64(mc.Raw),
				MhType:   mh.SHA2_256,
				MhLength: -1, // default length
			}
			contentCid, err := pref.Sum(fileContent)
			if err != nil {
				log.Error(err.Error())
				return
			}

			log.Info("Requested file, CID: ", req.Cid)
			log.Info("Requested file, length: ", req.TotalLength)
			log.Info("Received file, CID: ", contentCid)
			log.Info("Received file, length: ", len(fileContent))

			file, err := os.Create(filepath.Join(basePath, req.Cid))
			if err != nil {
				log.Error(err.Error())
				return
			}

			_, err = file.Write(fileContent)
			if err != nil {
				log.Error(err.Error())
				return
			}
		}
	}

	if err := s.CloseWrite(); err != nil {
		log.Error(err.Error())
		return
	}
}

func (ts *TransportServer) handleChunkInfo(req *types.FileChunkReq, path string) {
	ts.DbLk.Lock()
	defer ts.DbLk.Unlock()

	var fileInfo *types.ReceivedFileInfo
	key := datastore.NewKey(fmt.Sprintf("fileIno_%s", req.Cid))

	if req.ChunkId == 0 {
		fileInfo = &types.ReceivedFileInfo{
			Cid:            req.Cid,
			TotalLength:    req.TotalLength,
			TotalChunks:    req.TotalChunks,
			ReceivedLength: len(req.Content),
			Path:           path,
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
