package model

import (
	"context"
	"fmt"
	"sao-storage-node/node/cache"
	"sao-storage-node/node/config"
	"sao-storage-node/node/model/json_patch"
	"sao-storage-node/node/model/schema/validator"
	"sao-storage-node/node/storage"
	"sao-storage-node/types"
	"strings"
	"sync"

	cid "github.com/ipfs/go-cid"
	logging "github.com/ipfs/go-log/v2"
	jsoniter "github.com/json-iterator/go"
	"github.com/multiformats/go-multihash"
	"github.com/tendermint/tendermint/types/time"
	"golang.org/x/xerrors"
)

const PROPERTY_CONTEXT = "@context"
const PROPERTY_TYPE = "@type"
const MODEL_TYPE_FILE = "File"

var log = logging.Logger("model")

type Model struct {
	DataId     string
	Alias      string
	Tags       []string
	Content    []byte
	Cid        cid.Cid
	ExtendInfo string
}

type ModelManager struct {
	CacheCfg     *config.Cache
	CacheSvc     *cache.CacheSvc
	JsonpatchSvc *json_patch.JsonpatchSvc
	// used by gateway module
	CommitSvc storage.CommitSvcApi
}

var (
	modelManager *ModelManager
	once         sync.Once
)

func NewModelManager(cacheCfg *config.Cache, commitSvc storage.CommitSvcApi) *ModelManager {
	once.Do(func() {
		modelManager = &ModelManager{
			CacheCfg:     cacheCfg,
			CacheSvc:     cache.NewCacheSvc(),
			JsonpatchSvc: json_patch.NewJsonpatchSvc(),
			CommitSvc:    commitSvc,
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
	model := m.loadModel(account, key)
	if model != nil {
		return model, nil
	}

	result, err := m.CommitSvc.Pull(ctx, key)
	if err != nil {
		return nil, xerrors.Errorf(err.Error())
	}

	model = &Model{
		DataId:  result.DataId,
		Alias:   result.Alias,
		Content: result.Content,
		Cid:     result.Cid,
	}

	mm := model

	m.cacheModel(account, mm)

	return mm, nil
}

func (m *ModelManager) Create(ctx context.Context, orderMeta types.OrderMeta, content []byte) (*Model, error) {
	var alias string
	if orderMeta.Alias == "" {
		if orderMeta.Cid != cid.Undef {
			alias = orderMeta.Cid.String()
		} else if len(content) > 0 {
			alias = GenerateAlias(content)
		} else {
			alias = GenerateAlias([]byte(fmt.Sprintf("%d", time.Now().UnixNano())))
		}
		log.Debug("use a system generated alias ", alias)
		orderMeta.Alias = alias
	} else {
		alias = orderMeta.Alias
	}
	log.Debug("model alias ", orderMeta.Alias)

	oldModel := m.loadModel(orderMeta.Creator, orderMeta.Alias)
	if oldModel != nil {
		return nil, xerrors.Errorf("the model is exsiting already, alias: %s, dataId: %s", oldModel.Alias, oldModel.DataId)
	} else {
		log.Info("new model request")
	}

	err := m.validateModel(orderMeta.Creator, alias, content, orderMeta.Rule)
	if err != nil {
		log.Error(err.Error())
		return nil, xerrors.Errorf(err.Error())
	}

	// Commit
	orderMeta.CompleteTimeoutBlocks = 24 * 60 * 60
	result, err := m.CommitSvc.Commit(ctx, orderMeta.Creator, orderMeta, content)
	if err != nil {
		return nil, xerrors.Errorf(err.Error())
	}

	model := &Model{
		DataId:  result.DataId,
		Alias:   alias,
		Content: content,
		Cid:     orderMeta.Cid,
	}

	m.cacheModel(orderMeta.Creator, model)

	return model, nil
}

func (m *ModelManager) Update(account string, alias string, patch string, rule string) (*Model, error) {
	orgModel := m.loadModel(account, alias)
	if orgModel == nil {
		return nil, xerrors.Errorf("invalid model alias %s", alias)
	}

	newContentBytes, err := m.JsonpatchSvc.ApplyPatch(orgModel.Content, []byte(patch))
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
		DataId:  orgModel.DataId,
		Alias:   orgModel.Alias,
		Content: newContentBytes,
		Cid:     cid.NewCidV1(cid.Raw, multihash.Multihash(alias)),
	}

	m.cacheModel(account, model)

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
		if IsDataId(schema) {
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

func (m *ModelManager) loadModel(account string, key string) *Model {
	if !m.CacheCfg.EnableCache {
		return nil
	}

	value, err := m.CacheSvc.Get(account, key)
	if err != nil {
		if strings.Contains(err.Error(), fmt.Sprintf("the cache [%s] not found", account)) {
			err = m.CacheSvc.CreateCache(account, m.CacheCfg.CacheCapacity)
			if err != nil {
				log.Error(err.Error())
				return nil
			}
		} else {
			log.Error(err.Error())
			return nil
		}
	}

	if value != nil {
		if IsDataId(key) {
			return value.(*Model)
		} else {
			model, err := m.CacheSvc.Get(account, value.(string))
			if model != nil {
				return model.(*Model)
			}

			if err != nil {
				log.Warn(err.Error())
			}
		}
	}

	return nil
}

func (m *ModelManager) cacheModel(account string, model *Model) {
	if !m.CacheCfg.EnableCache {
		return
	}

	if len(model.Content) > m.CacheCfg.ContentLimit {
		m.CacheSvc.Put(account, model.DataId, model.Cid.String())
	} else {
		m.CacheSvc.Put(account, model.DataId, model)
	}

	m.CacheSvc.Put(account, model.Alias, model.DataId)
	// for _, k := range model.Tags {
	// 	m.CacheSvc.Put(account, k, model.DataId)
	// }
}
