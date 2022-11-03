package model

import (
	"context"
	"fmt"
	"regexp"
	"sao-storage-node/node/cache"
	"sao-storage-node/node/config"
	"sao-storage-node/node/gateway"
	"sao-storage-node/node/model/json_patch"
	"sao-storage-node/node/model/schema/validator"
	"sao-storage-node/node/utils"
	"sao-storage-node/types"
	"strings"
	"sync"

	cid "github.com/ipfs/go-cid"
	logging "github.com/ipfs/go-log/v2"
	jsoniter "github.com/json-iterator/go"
	"github.com/tendermint/tendermint/types/time"
	"golang.org/x/xerrors"
)

const PROPERTY_CONTEXT = "@context"
const PROPERTY_TYPE = "@type"
const MODEL_TYPE_FILE = "File"

var log = logging.Logger("model")

type ModelManager struct {
	CacheCfg     *config.Cache
	CacheSvc     cache.CacheSvcApi
	JsonpatchSvc *json_patch.JsonpatchSvc
	// used by gateway module
	GatewaySvc gateway.GatewaySvcApi
}

var (
	modelManager *ModelManager
	once         sync.Once
)

func NewModelManager(cacheCfg *config.Cache, gatewaySvc gateway.GatewaySvcApi) *ModelManager {
	once.Do(func() {
		var cacheSvc cache.CacheSvcApi
		if cacheCfg.RedisConn == "" {
			cacheSvc = cache.NewLruCacheSvc()
		} else {
			cacheSvc = cache.NewRedisCacheSvc(cacheCfg.RedisConn, cacheCfg.RedisPassword, cacheCfg.RedisPoolSize)
		}

		modelManager = &ModelManager{
			CacheCfg:     cacheCfg,
			CacheSvc:     cacheSvc,
			JsonpatchSvc: json_patch.NewJsonpatchSvc(),
			GatewaySvc:   gatewaySvc,
		}
	})

	return modelManager
}

func (mm *ModelManager) Stop(ctx context.Context) error {
	log.Info("stopping model manager...")

	mm.GatewaySvc.Stop(ctx)

	return nil
}

func (mm *ModelManager) Load(ctx context.Context, account string, key string, group string) (*types.Model, error) {
	meta, err := mm.GatewaySvc.QueryMeta(ctx, account, key, group)
	if err != nil {
		return nil, xerrors.Errorf(err.Error())
	}

	model := mm.loadModel(account, meta.DataId)
	if model != nil {
		if model.CommitId == meta.CommitId {
			return model, nil
		}
	}

	result, err := mm.GatewaySvc.FetchContent(ctx, meta)
	if err != nil {
		return nil, xerrors.Errorf(err.Error())
	}

	if model == nil {
		model = &types.Model{
			DataId:  meta.DataId,
			Alias:   meta.Alias,
			GroupId: meta.GroupId,
			Creator: meta.Creator,
			OrderId: meta.OrderId,
			Tags:    meta.Tags,
			// Cid: N/a,
			Shards:   meta.Shards,
			CommitId: meta.CommitId,
			Commits:  meta.Commits,
			// Content: N/a,
			ExtendInfo: meta.ExtendInfo,
		}
	}
	model.Cid = result.Cid
	model.Content = result.Content

	mm.cacheModel(account, model)

	return model, nil
}

func (mm *ModelManager) Create(ctx context.Context, orderMeta types.OrderMeta, content []byte) (*types.Model, error) {
	var alias string
	if orderMeta.Alias == "" {
		if orderMeta.Cid != cid.Undef {
			alias = orderMeta.Cid.String()
		} else if len(content) > 0 {
			alias = utils.GenerateAlias(content)
		} else {
			alias = utils.GenerateAlias([]byte(fmt.Sprintf("%d", time.Now().UnixNano())))
		}
		log.Debug("use a system generated alias ", alias)
		orderMeta.Alias = alias
	} else {
		alias = orderMeta.Alias
	}

	oldModel := mm.loadModel(orderMeta.Creator, orderMeta.Alias)
	if oldModel != nil {
		return nil, xerrors.Errorf("the model is exsiting already, alias: %s, dataId: %s", oldModel.Alias, oldModel.DataId)
	}

	err := mm.validateModel(orderMeta.Creator, alias, content, orderMeta.Rule)
	if err != nil {
		log.Error(err.Error())
		return nil, xerrors.Errorf(err.Error())
	}

	// Commit
	orderMeta.CompleteTimeoutBlocks = 24 * 60 * 60
	result, err := mm.GatewaySvc.CommitModel(ctx, orderMeta.Creator, orderMeta, content)
	if err != nil {
		return nil, xerrors.Errorf(err.Error())
	}

	model := &types.Model{
		DataId:  result.DataId,
		Alias:   alias,
		GroupId: orderMeta.GroupId,
		Creator: orderMeta.Creator,
		OrderId: result.OrderId,
		Tags:    orderMeta.Tags,
		// Cid: N/a,
		// Shards:   meta.Shards,
		CommitId:   result.CommitId,
		Commits:    append(make([]string, 0), result.CommitId),
		Content:    content,
		ExtendInfo: orderMeta.ExtenInfo,
	}

	mm.cacheModel(orderMeta.Creator, model)

	return model, nil
}

func (mm *ModelManager) Update(account string, alias string, patch string, rule string) (*types.Model, error) {
	orgModel := mm.loadModel(account, alias)
	if orgModel == nil {
		return nil, xerrors.Errorf("invalid model alias %s", alias)
	}

	newContentBytes, err := mm.JsonpatchSvc.ApplyPatch(orgModel.Content, []byte(patch))
	if err != nil {
		return nil, xerrors.Errorf(err.Error())
	}

	err = mm.validateModel(account, alias, newContentBytes, rule)
	if err != nil {
		return nil, xerrors.Errorf(err.Error())
	}

	// model, err = mm.CommitSvc.Commit(account, content)
	// if err != nil {
	// 	return nil, xerrors.Errorf(err.Error())
	// }
	model := &types.Model{
		DataId:  orgModel.DataId,
		Alias:   orgModel.Alias,
		Content: newContentBytes,
		// Cids:    make([]string, 1),
	}

	// model := &types.Model{
	// 	DataId:  result.DataId,
	// 	Alias:   alias,
	// 	GroupId: orderMeta.GroupId,
	// 	Creator: orderMeta.Creator,
	// 	OrderId: result.OrderId,
	// 	Tags:    orderMeta.Tags,
	// 	// Cid: N/a,
	// 	// Shards:   meta.Shards,
	// 	CommitId:   result.CommitId,
	// 	Commits:    append(make([]string, 0), result.CommitId),
	// 	Content:    content,
	// 	ExtendInfo: orderMeta.ExtenInfo,
	// }

	mm.cacheModel(account, model)

	return model, nil
}

func (mm *ModelManager) Delete(ctx context.Context, account string, key string, group string) (*types.Model, error) {
	model, _ := mm.CacheSvc.Get(account, key)
	if model != nil {
		m := model.(*types.Model)

		mm.CacheSvc.Evict(account, m.DataId)
		mm.CacheSvc.Evict(account, m.Alias)

		return &types.Model{
			DataId: m.DataId,
			Alias:  m.Alias,
		}, nil
	}

	return nil, nil
}

func (mm *ModelManager) validateModel(account string, alias string, contentBytes []byte, rule string) error {
	schemaStr := jsoniter.Get(contentBytes, PROPERTY_CONTEXT).ToString()
	if schemaStr == "" {
		return nil
	}

	match, err := regexp.Match(`^\[.*\]$`, []byte(schemaStr))
	if err != nil {
		return xerrors.Errorf(err.Error())
	}

	if match {
		schemas := []interface{}{}
		iter := jsoniter.ParseString(jsoniter.ConfigDefault, schemaStr)
		iter.ReadArrayCB(func(iter *jsoniter.Iterator) bool {
			var elem interface{}
			iter.ReadVal(&elem)
			schemas = append(schemas, elem)
			return true
		})

		for _, schema := range schemas {
			sch := schema.(string)
			if sch != "" {
				if utils.IsDataId(sch) {
					model, err := mm.CacheSvc.Get(account, sch)
					if err != nil {
						return xerrors.Errorf(err.Error())
					}
					sch = string(model.(*types.Model).Content)
				}

				validator, err := validator.NewDataModelValidator(alias, sch, rule)
				if err != nil {
					return xerrors.Errorf(err.Error())
				}
				err = validator.Validate(jsoniter.Get(contentBytes))
				if err != nil {
					return xerrors.Errorf(err.Error())
				}
			}
		}
	} else {
		iter := jsoniter.ParseString(jsoniter.ConfigDefault, schemaStr)
		dataId := iter.ReadString()
		var schema string
		if utils.IsDataId(dataId) {
			model, err := mm.CacheSvc.Get(account, dataId)
			if err != nil {
				return xerrors.Errorf(err.Error())
			}
			schema = string(model.(*types.Model).Content)
		} else {
			schema = iter.ReadObject()
		}

		validator, err := validator.NewDataModelValidator(alias, schema, rule)
		if err != nil {
			return xerrors.Errorf(err.Error())
		}
		err = validator.Validate(jsoniter.Get(contentBytes))
		if err != nil {
			return xerrors.Errorf(err.Error())
		}
	}

	return nil
}

func (mm *ModelManager) loadModel(account string, key string) *types.Model {
	if !mm.CacheCfg.EnableCache {
		return nil
	}

	value, err := mm.CacheSvc.Get(account, key)
	if err != nil {
		if strings.Contains(err.Error(), fmt.Sprintf("the cache [%s] not found", account)) {
			err = mm.CacheSvc.CreateCache(account, mm.CacheCfg.CacheCapacity)
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
		var model *types.Model
		if !utils.IsDataId(key) {
			value, err = mm.CacheSvc.Get(account, value.(string))
			if err != nil {
				log.Warn(err.Error())
			}
		}

		model = value.(*types.Model)
		if len(model.Content) == 0 && len(model.Shards) > 0 {
			log.Warnf("large size content should go through P2P channel")
		}
		return model
	}

	return nil
}

func (mm *ModelManager) cacheModel(account string, model *types.Model) {
	if !mm.CacheCfg.EnableCache {
		return
	}

	if len(model.Content) > mm.CacheCfg.ContentLimit {
		// large size content should go through P2P channel
		model.Content = make([]byte, 0)
	}
	mm.CacheSvc.Put(account, model.DataId, model)
	mm.CacheSvc.Put(account, model.Alias, model.DataId)

	// Reserved for open data model search feature...
	// for _, k := range model.Tags {
	// 	mm.CacheSvc.Put(account, k, model.DataId)
	// }
}
