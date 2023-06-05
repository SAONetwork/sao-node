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
	"sao-node/api"
	"sao-node/node/config"
	"sao-node/types"
	"sao-node/utils"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p"
	libp2pwebtransport "github.com/libp2p/go-libp2p/p2p/transport/webtransport"
	"github.com/mitchellh/go-homedir"
	ma "github.com/multiformats/go-multiaddr"

	"github.com/ipfs/go-datastore"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/network"
)

type Libp2pRpcServer struct {
	Ctx              context.Context
	DbLk             sync.Mutex
	Db               datastore.Batching
	GatewayApi       api.SaoApi
	StagingPath      string
	StagingSapceSize int64
}

func StartLibp2pRpcServer(ctx context.Context, ga api.SaoApi, address string, serverKey crypto.PrivKey, db datastore.Batching, cfg *config.Node, stagingPath string) (*Libp2pRpcServer, error) {
	tr, err := libp2pwebtransport.New(serverKey, nil, network.NullResourceManager)
	if err != nil {
		return nil, err
	}

	h, err := libp2p.New(libp2p.Transport(tr), libp2p.Identity(serverKey))
	if err != nil {
		return nil, err
	}

	err = h.Network().Listen(ma.StringCast(address + "/quic/webtransport"))
	if err != nil {
		return nil, err
	}

	var peerInfos []string
	for _, a := range h.Addrs() {
		withP2p := a.Encapsulate(ma.StringCast("/p2p/" + h.ID().String()))
		log.Debug("addr=", withP2p.String())
		peerInfos = append(peerInfos, withP2p.String())
	}
	if len(peerInfos) > 0 {
		key := datastore.NewKey(fmt.Sprintf(types.PEER_INFO_PREFIX))
		peers, err := db.Get(ctx, key)
		if err != nil {
			return nil, err
		}
		if len(peers) > 0 {
			db.Put(ctx, key, []byte(string(peers)+","+strings.Join(peerInfos, ",")))
		} else {
			db.Put(ctx, key, []byte(strings.Join(peerInfos, ",")))
		}
	}

	rs := &Libp2pRpcServer{
		Ctx:              ctx,
		Db:               db,
		GatewayApi:       ga,
		StagingPath:      stagingPath,
		StagingSapceSize: cfg.Transport.StagingSapceSize,
	}

	h.Network().SetStreamHandler(rs.HandleStream)

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	select {
	case <-c:
	case <-time.After(time.Second):
	}

	return rs, nil
}

func (rs *Libp2pRpcServer) HandleStream(s network.Stream) {
	defer s.Close()

	// Set a deadline on reading from the stream so it doesn’t hang
	_ = s.SetReadDeadline(time.Now().Add(30 * time.Second))
	defer s.SetReadDeadline(time.Time{}) // nolint

	var req types.RpcReq
	var resp = types.RpcResp{}

	buf := &bytes.Buffer{}
	buf.ReadFrom(s)
	err := json.Unmarshal(buf.Bytes(), &req)
	if err == nil {
		log.Info("Got rpc request: ", req.Method)

		var result string
		var err error
		switch req.Method {
		case "Sao.Upload":
			req.Params = append(req.Params, filepath.Join(rs.StagingPath, s.Conn().RemotePeer().String()))
			result, err = rs.upload(req.Params)
		case "Sao.ModelCreate":
			result, err = rs.create(req.Params)
		case "Sao.ModelLoad":
			result, err = rs.load(req.Params)
		case "Sao.ModelUpdate":
			result, err = rs.update(req.Params)
		default:
			resp.Error = "N/a"
		}
		if err != nil {
			resp.Error = err.Error()
		} else {
			resp.Data = result
		}

	} else {
		resp.Error = err.Error()
	}

	bytes, err := json.Marshal(resp)
	if err != nil {
		log.Error(err.Error())
		return
	}

	if _, err := s.Write(bytes); err != nil {
		log.Error(err.Error())
		return
	}

	if err := s.CloseWrite(); err != nil {
		log.Error(err.Error())
		return
	}

	log.Info("Sent rpc response: ", resp)
}

func (rs *Libp2pRpcServer) handleChunkInfo(req *types.FileChunkReq, path string) {
	rs.DbLk.Lock()
	defer rs.DbLk.Unlock()

	var fileInfo *types.ReceivedFileInfo
	key := datastore.NewKey(types.FILE_INFO_PREFIX + req.Cid)

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
	} else if info, err := rs.Db.Get(rs.Ctx, key); err == nil {
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

	err = rs.Db.Put(rs.Ctx, key, info)
	if err != nil {
		log.Error(err.Error())
		return
	}
}

func (rs *Libp2pRpcServer) upload(params []string) (string, error) {
	if len(params) != 2 {
		return "", types.Wrapf(types.ErrInvalidParameters, "invalid params length")
	}

	var req types.FileChunkReq
	err := json.Unmarshal([]byte(params[0]), &req)
	if err != nil {
		log.Error(err.Error())
		return "", err
	}

	localCid, err := utils.CalculateCid(req.Content)
	if err != nil {
		return "", err
	}

	if len(req.Content) > 0 {
		stagingPath, err := homedir.Expand(rs.StagingPath)
		if err != nil {
			return "", err
		}

		info, err := os.Stat(stagingPath)
		if os.IsNotExist(err) {
			err = os.MkdirAll(stagingPath, 0700)
			if err != nil {
				return "", err
			}

			if int64(len(req.Content)) > rs.StagingSapceSize {
				return "", types.Wrapf(types.ErrInvalidParameters, "not enough staging space under %s, need %v but only %v left", rs.StagingPath, len(req.Content), rs.StagingSapceSize-info.Size())
			}
		} else if err != nil {
			return "", err
		} else {
			if info.Size()+int64(len(req.Content)) > rs.StagingSapceSize {
				return "", types.Wrapf(types.ErrInvalidParameters, "not enough staging space under %s, need %v but only %v left", rs.StagingPath, len(req.Content), rs.StagingSapceSize-info.Size())
			}
		}

		path := filepath.Join(params[1], req.Cid)
		rs.handleChunkInfo(&req, path)

		path, err = homedir.Expand(path)
		if err != nil {
			return "", err
		}
		log.Info("path: ", path)
		_, err = os.Stat(path)
		if err != nil {
			if !os.IsExist(err) {
				err = os.MkdirAll(path, 0755)
				if err != nil && !os.IsExist(err) {
					return "", err
				}
			} else {
				return "", err
			}
		}

		file, err := os.Create(filepath.Join(path, req.ChunkCid))
		if err != nil {
			return "", err
		}

		_, err = file.Write(req.Content)
		if err != nil {
			return "", err
		}

		log.Infof("Received file chunk[%d], remote CID: %s, local CID: %s", req.ChunkId, req.ChunkCid, localCid)
		log.Infof("Staging file %s generated", filepath.Join(path, req.ChunkCid))
	} else {
		// Transport is done
		key := datastore.NewKey(types.FILE_INFO_PREFIX + req.Cid)
		if info, err := rs.Db.Get(rs.Ctx, key); err == nil {
			var fileInfo *types.ReceivedFileInfo
			err := json.Unmarshal(info, &fileInfo)
			if err != nil {
				return "", err
			}

			basePath, err := homedir.Expand(fileInfo.Path)
			if err != nil {
				return "", err
			}
			log.Info("path: ", basePath)

			var fileContent []byte
			for _, chunkCid := range fileInfo.ChunkCids {
				var path = filepath.Join(basePath, chunkCid)
				file, err := os.Open(path)
				if err != nil {
					return "", err
				}

				content, err := io.ReadAll(file)
				if err != nil {
					return "", err
				}

				fileContent = append(fileContent, content...)
			}

			contentCid, err := utils.CalculateCid(fileContent)
			if err != nil {
				return "", err
			}

			log.Info("Requested file, CID: ", req.Cid)
			log.Info("Requested file, length: ", req.TotalLength)
			log.Info("Received file, CID: ", contentCid)
			log.Info("Received file, length: ", len(fileContent))

			file, err := os.Create(filepath.Join(basePath, req.Cid))
			if err != nil {
				return "", err
			}

			_, err = file.Write(fileContent)
			if err != nil {
				return "", err
			}
		} else {
			return "", err
		}
	}

	return req.Cid, nil
}

func (rs *Libp2pRpcServer) create(params []string) (string, error) {
	if len(params) != 3 {
		return "", types.Wrapf(types.ErrInvalidParameters, "invalid params length")
	}

	var req types.MetadataProposal
	err := json.Unmarshal([]byte(params[0]), &req)
	if err != nil {
		log.Error(err.Error())
		return "", nil
	}

	var orderProposal types.OrderStoreProposal
	err = json.Unmarshal([]byte(params[1]), &orderProposal)
	if err != nil {
		log.Error(err.Error())
		return "", nil
	}

	orderId, err := strconv.ParseInt(params[2], 10, 64)
	if err != nil {
		return "", types.Wrap(types.ErrInvalidParameters, err)
	}

	resp, err := rs.GatewayApi.ModelCreate(rs.Ctx, &req, &orderProposal, uint64(orderId), []byte(params[2]))
	if err != nil {
		log.Error(err.Error())
		return "", nil
	}
	b, err := json.Marshal(resp)
	if err != nil {
		log.Error(err.Error())
		return "", nil
	}
	return string(b), nil
}

func (rs *Libp2pRpcServer) load(params []string) (string, error) {
	if len(params) != 1 {
		return "", types.Wrapf(types.ErrInvalidParameters, "invalid params length")
	}

	var req types.MetadataProposal
	err := json.Unmarshal([]byte(params[0]), &req)
	if err != nil {
		log.Error(err.Error())
		return "", nil
	}
	resp, err := rs.GatewayApi.ModelLoad(rs.Ctx, &req)
	if err != nil {
		log.Error(err.Error())
		return "", nil
	}
	b, err := json.Marshal(resp)
	if err != nil {
		log.Error(err.Error())
		return "", nil
	}
	return string(b), nil
}

func (rs *Libp2pRpcServer) update(params []string) (string, error) {
	if len(params) != 3 {
		return "", types.Wrapf(types.ErrInvalidParameters, "invalid params length")
	}

	var req types.MetadataProposal
	err := json.Unmarshal([]byte(params[0]), &req)
	if err != nil {
		log.Error(err.Error())
		return "", nil
	}

	var orderProposal types.OrderStoreProposal
	err = json.Unmarshal([]byte(params[1]), &orderProposal)
	if err != nil {
		log.Error(err.Error())
		return "", nil
	}

	orderId, err := strconv.ParseInt(params[2], 10, 64)
	if err != nil {
		return "", types.Wrap(types.ErrInvalidParameters, err)
	}

	resp, err := rs.GatewayApi.ModelUpdate(rs.Ctx, &req, &orderProposal, uint64(orderId), []byte(params[2]))
	if err != nil {
		log.Error(err.Error())
		return "", nil
	}
	b, err := json.Marshal(resp)
	if err != nil {
		log.Error(err.Error())
		return "", nil
	}
	return string(b), nil
}
