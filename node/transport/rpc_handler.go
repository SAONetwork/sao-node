package transport

import (
	"context"
	"encoding/json"
	sidtypes "github.com/SaoNetwork/sao/x/did/types"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"sync"

	"github.com/SaoNetwork/sao-node/api"
	"github.com/SaoNetwork/sao-node/node/config"
	"github.com/SaoNetwork/sao-node/types"
	"github.com/SaoNetwork/sao-node/utils"
	"github.com/ipfs/go-datastore"
	"github.com/mitchellh/go-homedir"
)

type RpcHandler struct {
	Ctx              context.Context
	DbLk             sync.Mutex
	Db               datastore.Batching
	GatewayApi       api.SaoApi
	StagingPath      string
	StagingSpaceSize int64
}

func NewHandler(ctx context.Context, ga api.SaoApi, db datastore.Batching, cfg *config.Node, stagingPath string) *RpcHandler {

	handler := RpcHandler{
		Ctx:              ctx,
		Db:               db,
		GatewayApi:       ga,
		StagingPath:      stagingPath,
		StagingSpaceSize: cfg.Transport.StagingSpaceSize,
	}
	return &handler
}

func (rs *RpcHandler) handleChunkInfo(req *types.FileChunkReq, path string) {
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

func (rs *RpcHandler) Upload(params []string) (string, error) {
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

			if int64(len(req.Content)) > rs.StagingSpaceSize {
				return "", types.Wrapf(types.ErrInvalidParameters, "not enough staging space under %s, need %v but only %v left", rs.StagingPath, len(req.Content), rs.StagingSpaceSize-info.Size())
			}
		} else if err != nil {
			return "", err
		} else {
			if info.Size()+int64(len(req.Content)) > rs.StagingSpaceSize {
				return "", types.Wrapf(types.ErrInvalidParameters, "not enough staging space under %s, need %v but only %v left", rs.StagingPath, len(req.Content), rs.StagingSpaceSize-info.Size())
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

	return localCid.String(), nil
}

func (rs *RpcHandler) Create(params []string) (string, error) {
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

func (rs *RpcHandler) Load(params []string) (string, error) {
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

func (rs *RpcHandler) Update(params []string) (string, error) {
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

func (rs *RpcHandler) BindingProof(params []string) (string, error) {
	if len(params) != 3 {
		return "", types.Wrapf(types.ErrInvalidParameters, "invalid params length")
	}

	var keys []*sidtypes.PubKey
	err := json.Unmarshal([]byte(params[1]), &keys)
	if err != nil {
		log.Error(err.Error())
		return "", nil
	}

	var accAuth sidtypes.AccountAuth
	err = json.Unmarshal([]byte(params[2]), &accAuth)
	if err != nil {
		log.Error(err.Error())
		return "", nil
	}

	var proof sidtypes.BindingProof
	err = json.Unmarshal([]byte(params[3]), &proof)
	if err != nil {
		log.Error(err.Error())
		return "", nil
	}

	resp, err := rs.GatewayApi.DidBindingProof(rs.Ctx, params[0], keys, &accAuth, &proof)
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
