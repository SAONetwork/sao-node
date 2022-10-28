package model

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sao-storage-node/node/cache"
	"sao-storage-node/node/config"
	"sao-storage-node/node/model/json_patch"
	"sao-storage-node/node/model/schema/validator"
	"sao-storage-node/node/storage"
	"sao-storage-node/types"
	"sao-storage-node/types/model"
	"sao-storage-node/types/transport"
	"strings"
	"sync"

	"github.com/google/uuid"
	mc "github.com/multiformats/go-multicodec"

	cid "github.com/ipfs/go-cid"
	"github.com/ipfs/go-datastore"
	logging "github.com/ipfs/go-log/v2"
	jsoniter "github.com/json-iterator/go"
	"github.com/mitchellh/go-homedir"
	"github.com/multiformats/go-multihash"
	"github.com/tendermint/tendermint/types/time"
	"golang.org/x/xerrors"
)

const PROPERTY_CONTEXT = "@context"
const PROPERTY_TYPE = "@type"
const MODEL_TYPE_FILE = "File"

var log = logging.Logger("model")

type Model struct {
	DataId  string
	Alias   string
	Tags    []string
	Schema  []byte
	Type    types.ModelType
	Content []byte
	OrderId uint64
	Cid     cid.Cid
}

type ModelManager struct {
	CacheCfg     *config.Cache
	CacheSvc     *cache.CacheSvc
	JsonpatchSvc *json_patch.JsonpatchSvc
	// used by gateway module
	CommitSvc *storage.CommitSvc
	Db        datastore.Batching
}

var (
	modelManager *ModelManager
	once         sync.Once
)

func NewModelManager(cacheCfg *config.Cache, commitSvc *storage.CommitSvc, db datastore.Batching) *ModelManager {
	once.Do(func() {
		modelManager = &ModelManager{
			CacheCfg:     cacheCfg,
			CacheSvc:     cache.NewCacheSvc(),
			JsonpatchSvc: json_patch.NewJsonpatchSvc(),
			CommitSvc:    commitSvc,
			Db:           db,
		}
	})

	return modelManager
}

func (m *ModelManager) Stop(ctx context.Context) error {
	if m.CommitSvc != nil {
		m.CommitSvc.Stop(ctx)
	}
	return nil
}

func (m *ModelManager) Load(ctx context.Context, account string, key string) (*Model, error) {
	model, err := m.CacheSvc.Get(account, key)
	if model != nil {
		log.Info("model: ", model)

		return model.(*Model), nil
	}

	if err != nil {
		if strings.Contains(err.Error(), fmt.Sprintf("the cache [%s] not found", account)) {
			err = m.CacheSvc.CreateCache(account, m.CacheCfg.CacheCapacity)
			if err != nil {
				log.Error(err.Error())
				return nil, xerrors.Errorf(err.Error())
			}
		} else {
			return nil, xerrors.Errorf(err.Error())
		}
	}

	result, err := m.CommitSvc.Pull(ctx, key)
	if err != nil {
		return nil, xerrors.Errorf(err.Error())
	}

	model = &Model{
		DataId:  result.DataId,
		Alias:   result.Alias,
		Content: result.Content,
		Type:    result.Type,
		Cid:     result.Cid,
		OrderId: result.OrderId,
	}

	mm := model.(*Model)

	m.cacheModel(account, mm.Alias, mm)

	return mm, nil
}

func (m *ModelManager) Create(ctx context.Context, orderMeta types.OrderMeta, modelType types.ModelType) (*Model, error) {
	var alias string
	if orderMeta.Alias == "" {
		if orderMeta.Cid != cid.Undef {
			alias = orderMeta.Cid.String()
		} else if len(orderMeta.Content) > 0 {
			alias = GenerateAlias(orderMeta.Content)
		} else {
			alias = GenerateAlias([]byte(fmt.Sprintf("%d", time.Now().UnixNano())))
		}
		log.Info("use a system generated alias ", alias)
		orderMeta.Alias = alias
	} else {
		alias = orderMeta.Alias
	}
	log.Info("model alias ", orderMeta.Alias)

	oldModel, err := m.CacheSvc.Get(orderMeta.Creator, orderMeta.Alias)
	if err != nil {
		if strings.Contains(err.Error(), fmt.Sprintf("the cache [%s] not found", orderMeta.Creator)) {
			err = m.CacheSvc.CreateCache(orderMeta.Creator, m.CacheCfg.CacheCapacity)
			if err != nil {
				return nil, xerrors.Errorf(err.Error())
			}
		} else {
			return nil, xerrors.Errorf(err.Error())
		}
	}
	if oldModel != nil {
		return nil, xerrors.Errorf("the model is exsiting already, alias: %s, dataId: %s", oldModel.(*Model).Alias, oldModel.(*Model).DataId)
	} else {
		log.Info("new model request")
	}

	if orderMeta.Cid != cid.Undef && len(orderMeta.Content) == 0 {
		// Asynchronous order and the content has been uploaded already
		key := datastore.NewKey(fmt.Sprintf("fileIno_%s", orderMeta.Cid))
		if info, err := m.Db.Get(ctx, key); err == nil {
			var fileInfo *transport.ReceivedFileInfo
			err := json.Unmarshal(info, &fileInfo)
			if err != nil {
				return nil, xerrors.Errorf(err.Error())
			}

			basePath, err := homedir.Expand(fileInfo.Path)
			if err != nil {
				return nil, xerrors.Errorf(err.Error())
			}
			log.Info("path: ", basePath)

			var path = filepath.Join(basePath, orderMeta.Cid.String())
			file, err := os.Open(path)
			if err != nil {
				return nil, xerrors.Errorf(err.Error())
			}

			content, err := io.ReadAll(file)
			if err != nil {
				return nil, xerrors.Errorf(err.Error())
			}
			orderMeta.Content = content
		} else {
			return nil, xerrors.Errorf("invliad CID: %s", orderMeta.Cid.String())
		}
	}

	var modelBytes []byte
	if modelType == types.ModelTypeFile {
		model := &model.FileModel{
			FileName: orderMeta.Alias,
			Tags:     orderMeta.Tags,
			Cid:      orderMeta.Cid.String(),
			Content:  orderMeta.Content,
		}
		modelBytes, err = json.Marshal(model)
		if err != nil {
			return nil, xerrors.Errorf(err.Error())
		}
	} else {
		modelBytes = orderMeta.Content
	}

	pref := cid.Prefix{
		Version:  1,
		Codec:    uint64(mc.Raw),
		MhType:   multihash.SHA2_256,
		MhLength: -1, // default length
	}
	modelCid, err := pref.Sum(modelBytes)
	if err != nil {
		return nil, xerrors.Errorf(err.Error())
	}
	orderMeta.Cid = modelCid

	err = m.validateModel(orderMeta.Creator, alias, modelBytes, orderMeta.Rule)
	if err != nil {
		log.Error(err.Error())
		return nil, xerrors.Errorf(err.Error())
	}

	// Commit
	orderMeta.CompleteTimeoutBlocks = 24 * 60 * 60
	result, err := m.CommitSvc.Commit(ctx, orderMeta.Creator, orderMeta, modelBytes)
	if err != nil {
		return nil, xerrors.Errorf(err.Error())
	}

	model := &Model{
		DataId:  result.DataId,
		Alias:   alias,
		Content: modelBytes,
		Type:    modelType,
		Cid:     orderMeta.Cid,
		OrderId: result.OrderId,
	}

	m.cacheModel(orderMeta.Creator, model.Alias, model)

	return model, nil
}

func (m *ModelManager) Update(account string, alias string, patch string, rule string) (*Model, error) {
	orgModel, err := m.CacheSvc.Get(account, alias)
	if err != nil || orgModel == nil {
		return nil, xerrors.Errorf(err.Error())
	}

	newContentBytes, err := m.JsonpatchSvc.ApplyPatch(orgModel.(*Model).Content, []byte(patch))
	if err != nil {
		return nil, xerrors.Errorf(err.Error())
	}

	err = m.validateModel(account, alias, newContentBytes, rule)
	if err != nil {
		return nil, xerrors.Errorf(err.Error())
	}

	// model, err = m.CommitSvc.Commit(account, content)
	// if err != nil {
	// 	return nil, xerrors.Errorf(err.Error())
	// }
	model := &Model{
		DataId:  orgModel.(*Model).DataId,
		Alias:   orgModel.(*Model).Alias,
		Content: newContentBytes,
		Cid:     cid.NewCidV1(cid.Raw, multihash.Multihash(alias)),
	}

	m.cacheModel(account, alias, model)

	return model, nil
}

func (mm *ModelManager) Delete(ctx context.Context, account string, key string) (*Model, error) {
	model, _ := mm.CacheSvc.Get(account, key)
	if model != nil {
		m := model.(*Model)

		mm.CacheSvc.Evict(account, m.DataId)
		mm.CacheSvc.Evict(account, m.Alias)

		return &Model{
			DataId: m.DataId,
			Alias:  m.Alias,
		}, nil
	}

	return nil, nil
}

func (m *ModelManager) validateModel(account string, alias string, contentBytes []byte, rule string) error {
	schema := jsoniter.Get(contentBytes, PROPERTY_CONTEXT).ToString()

	if schema != "" {
		_, err := uuid.Parse(schema)
		if err == nil {
			model, err := m.CacheSvc.Get(account, schema)
			if err != nil {
				return xerrors.Errorf(err.Error())
			}
			schema = string(model.(*Model).Content)
		}
	}

	validator, err := validator.NewDataModelValidator(alias, schema, rule)
	if err != nil {
		return xerrors.Errorf(err.Error())
	}
	err = validator.Validate(jsoniter.Get(contentBytes))
	if err != nil {
		return xerrors.Errorf(err.Error())
	}

	return nil
}

func (m *ModelManager) cacheModel(account string, alias string, model *Model) {
	if len(model.Content) > m.CacheCfg.ContentLimit {
		m.CacheSvc.Put(account, alias, model.Cid.String())
		m.CacheSvc.Put(account, model.DataId, model.Cid.String())
		for _, k := range model.Tags {
			m.CacheSvc.Put(account, k, model.Cid.String())
		}
	} else {
		m.CacheSvc.Put(account, alias, model)
		m.CacheSvc.Put(account, model.DataId, model)
		for _, k := range model.Tags {
			m.CacheSvc.Put(account, k, model)
		}
	}

	m.CacheSvc.Put(account, fmt.Sprintf("%d", model.OrderId), alias)
}
