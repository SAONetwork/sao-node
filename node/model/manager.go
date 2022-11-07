package model

import (
	"context"
	"fmt"
	"regexp"
	"sao-storage-node/node/cache"
	"sao-storage-node/node/config"
	"sao-storage-node/node/gateway"
	"sao-storage-node/node/model/schema/validator"
	"sao-storage-node/types"
	"sao-storage-node/utils"
	"strconv"
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
	CacheCfg *config.Cache
	CacheSvc cache.CacheSvcApi
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
			CacheCfg:   cacheCfg,
			CacheSvc:   cacheSvc,
			GatewaySvc: gatewaySvc,
		}
	})

	return modelManager
}

func (mm *ModelManager) Stop(ctx context.Context) error {
	log.Info("stopping model manager...")

	mm.GatewaySvc.Stop(ctx)

	return nil
}

func (mm *ModelManager) Load(ctx context.Context, orderMeta types.OrderMeta) (*types.Model, error) {
	key := orderMeta.DataId
	if !utils.IsDataId(key) {
		value, err := mm.CacheSvc.Get(orderMeta.Owner, orderMeta.Alias+orderMeta.GroupId)
		if err != nil {
			log.Warn(err.Error())
		} else if value != nil {
			dataId, ok := value.(string)
			if ok && utils.IsDataId(dataId) {
				key = dataId
			}
		}
	}

	meta, err := mm.GatewaySvc.QueryMeta(ctx, orderMeta.Owner, key, orderMeta.GroupId, 0)
	if err != nil {
		return nil, xerrors.Errorf(err.Error())
	}
	orderMeta.Alias = meta.Alias
	orderMeta.DataId = meta.DataId

	if orderMeta.Version != "" {
		index, err := strconv.Atoi(strings.ReplaceAll(orderMeta.Version, "v", ""))
		if err != nil {
			return nil, xerrors.Errorf(err.Error())
		}

		if len(meta.Commits) > index {
			commit := meta.Commits[index]
			commitInfo := strings.Split(meta.Commits[index], "\026")
			if len(commitInfo) != 2 || len(commitInfo[1]) == 0 {
				return nil, xerrors.Errorf("invalid commit information: %s", commit)
			}
			height, err := strconv.ParseInt(commitInfo[1], 10, 64)
			if err != nil {
				return nil, xerrors.Errorf(err.Error())
			}
			meta, err = mm.GatewaySvc.QueryMeta(ctx, orderMeta.Owner, key, orderMeta.GroupId, height)
			if err != nil {
				return nil, xerrors.Errorf(err.Error())
			}
		}
	}

	if orderMeta.CommitId != "" {
		for _, commit := range meta.Commits {
			if strings.HasPrefix(commit, orderMeta.CommitId) {
				commitInfo := strings.Split(commit, "\026")
				if len(commitInfo) != 2 || len(commitInfo[1]) == 0 {
					return nil, xerrors.Errorf("invalid commit information: %s", commit)
				}
				height, err := strconv.ParseInt(commitInfo[1], 10, 64)
				if err != nil {
					return nil, xerrors.Errorf(err.Error())
				}
				meta, err = mm.GatewaySvc.QueryMeta(ctx, orderMeta.Owner, key, orderMeta.GroupId, height)
				if err != nil {
					return nil, xerrors.Errorf(err.Error())
				}
				break
			}
		}
	}

	model := mm.loadModel(orderMeta.Owner, meta.DataId)
	if model != nil {
		if model.CommitId == meta.CommitId && len(model.Content) > 0 {
			return model, nil
		}
	}
	if model == nil {
		model = &types.Model{
			DataId:   meta.DataId,
			Alias:    meta.Alias,
			GroupId:  meta.GroupId,
			OrderId:  meta.OrderId,
			Owner:    meta.Owner,
			Tags:     meta.Tags,
			Cid:      meta.Cid,
			Shards:   meta.Shards,
			CommitId: meta.CommitId,
			Commits:  meta.Commits,
			// Content: N/a,
			ExtendInfo: meta.ExtendInfo,
		}
	}

	if len(meta.Shards) > 1 {
		log.Warnf("large size content should go through P2P channel")
	} else {
		result, err := mm.GatewaySvc.FetchContent(ctx, meta)
		if err != nil {
			return nil, xerrors.Errorf(err.Error())
		}
		model.Cid = result.Cid
		model.Content = result.Content
	}

	mm.cacheModel(orderMeta.Owner, model)

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

	oldModel := mm.loadModel(orderMeta.Owner, orderMeta.Alias)
	if oldModel != nil {
		return nil, xerrors.Errorf("the model is exsiting already, alias: %s, dataId: %s", oldModel.Alias, oldModel.DataId)
	}

	err := mm.validateModel(orderMeta.Owner, alias, content, orderMeta.Rule)
	if err != nil {
		log.Error(err.Error())
		return nil, xerrors.Errorf(err.Error())
	}

	// Commit
	orderMeta.CompleteTimeoutBlocks = 24 * 60 * 60
	result, err := mm.GatewaySvc.CommitModel(ctx, orderMeta.Owner, orderMeta, content)
	if err != nil {
		return nil, xerrors.Errorf(err.Error())
	}

	model := &types.Model{
		DataId:     result.DataId,
		Alias:      alias,
		GroupId:    orderMeta.GroupId,
		OrderId:    result.OrderId,
		Owner:      orderMeta.Owner,
		Tags:       orderMeta.Tags,
		Cid:        result.Cid,
		Shards:     result.Shards,
		CommitId:   result.CommitId,
		Commits:    append(make([]string, 0), result.CommitId),
		Content:    content,
		ExtendInfo: orderMeta.ExtendInfo,
	}

	mm.cacheModel(orderMeta.Owner, model)

	return model, nil
}

func (mm *ModelManager) Update(ctx context.Context, orderMeta types.OrderMeta, patch []byte) (*types.Model, error) {
	var key string
	if orderMeta.DataId == "" {
		key = orderMeta.Alias
	} else {
		key = orderMeta.DataId
	}
	meta, err := mm.GatewaySvc.QueryMeta(ctx, orderMeta.Owner, key, orderMeta.GroupId, 0)
	if err != nil {
		return nil, xerrors.Errorf(err.Error())
	}

	orderMeta.DataId = meta.DataId
	orderMeta.Alias = meta.Alias

	var isFetch = true
	orgModel := mm.loadModel(orderMeta.Owner, meta.DataId)
	if orgModel != nil {
		if orgModel.CommitId == meta.CommitId && len(orgModel.Content) > 0 {
			// found latest data model in local cache
			log.Debugf("load the model[%s]-%s from cache", meta.DataId, meta.Alias)
			log.Debugf("model: ", string(orgModel.Content))
			isFetch = false
		}
	} else {
		orgModel = &types.Model{
			DataId:   meta.DataId,
			Alias:    meta.Alias,
			GroupId:  meta.GroupId,
			OrderId:  meta.OrderId,
			Owner:    meta.Owner,
			Tags:     meta.Tags,
			Cid:      meta.Cid,
			Shards:   meta.Shards,
			CommitId: meta.CommitId,
			Commits:  meta.Commits,
			// Content: N/a,
			ExtendInfo: meta.ExtendInfo,
		}
	}

	if isFetch {
		result, err := mm.GatewaySvc.FetchContent(ctx, meta)
		if err != nil {
			return nil, xerrors.Errorf(err.Error())
		}
		log.Info("result: ", result)
		log.Info("orgModel: ", orgModel)
		orgModel.Content = result.Content
	}

	newContent, err := utils.ApplyPatch(orgModel.Content, []byte(patch))
	if err != nil {
		return nil, xerrors.Errorf(err.Error())
	}
	log.Debugf("newContent: ", string(newContent))
	log.Debugf("orgModel: ", string(orgModel.Content))

	newContentCid, err := utils.CaculateCid(newContent)
	if err != nil {
		return nil, xerrors.Errorf(err.Error())
	}
	if newContentCid != orderMeta.Cid {
		return nil, xerrors.Errorf("cid mismatch, expected %s, but got %s", orderMeta.Cid, newContentCid)
	}

	err = mm.validateModel(orderMeta.Owner, orderMeta.Alias, newContent, orderMeta.Rule)
	if err != nil {
		log.Error(err.Error())
		return nil, xerrors.Errorf(err.Error())
	}

	// Commit
	orderMeta.CompleteTimeoutBlocks = 24 * 60 * 60
	result, err := mm.GatewaySvc.CommitModel(ctx, orderMeta.Owner, orderMeta, newContent)
	if err != nil {
		return nil, xerrors.Errorf(err.Error())
	}

	model := &types.Model{
		DataId:     orderMeta.DataId,
		Alias:      orderMeta.Alias,
		GroupId:    orderMeta.GroupId,
		OrderId:    result.OrderId,
		Owner:      orderMeta.Owner,
		Tags:       orderMeta.Tags,
		Cid:        result.Cid,
		Shards:     result.Shards,
		CommitId:   result.CommitId,
		Commits:    append(meta.Commits, result.CommitId),
		Content:    newContent,
		ExtendInfo: orderMeta.ExtendInfo,
	}

	mm.cacheModel(orderMeta.Owner, model)

	return model, nil
}

func (mm *ModelManager) Delete(ctx context.Context, account string, key string, group string) (*types.Model, error) {
	meta, err := mm.GatewaySvc.QueryMeta(ctx, account, key, group, 0)
	if err != nil {
		return nil, xerrors.Errorf(err.Error())
	}

	model, _ := mm.CacheSvc.Get(account, meta.DataId)
	if model != nil {
		m, ok := model.(*types.Model)
		if ok {
			mm.CacheSvc.Evict(account, m.DataId)
			mm.CacheSvc.Evict(account, m.Alias+m.GroupId)

			return &types.Model{
				DataId: m.DataId,
				Alias:  m.Alias,
			}, nil
		}
	}

	return nil, nil
}

func (mm *ModelManager) ShowCommits(ctx context.Context, account string, key string, group string) (*types.Model, error) {
	if !utils.IsDataId(key) {
		value, err := mm.CacheSvc.Get(account, key+group)
		if err != nil {
			log.Warn(err.Error())
		} else if value != nil {
			dataId, ok := value.(string)
			if ok && utils.IsDataId(dataId) {
				key = dataId
			}
		}
	}
	meta, err := mm.GatewaySvc.QueryMeta(ctx, account, key, group, 0)
	if err != nil {
		return nil, xerrors.Errorf(err.Error())
	}

	return &types.Model{
		DataId:  meta.DataId,
		Alias:   meta.Alias,
		Commits: meta.Commits,
	}, nil
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
			sch, ok := schema.(string)
			if ok && sch != "" {
				if utils.IsDataId(sch) {
					model, err := mm.CacheSvc.Get(account, sch)
					if err != nil {
						return xerrors.Errorf(err.Error())
					}
					m, ok := model.(*types.Model)
					if ok {
						sch = string(m.Content)
					} else {
						return xerrors.Errorf("invalid schema: %v", m)
					}
				}

				validator, err := validator.NewDataModelValidator(alias, sch, rule)
				if err != nil {
					return xerrors.Errorf(err.Error())
				}
				err = validator.Validate(jsoniter.Get(contentBytes))
				if err != nil {
					return xerrors.Errorf(err.Error())
				}
			} else {
				return xerrors.Errorf("invalid schema: %v", schema)
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
			m, ok := model.(*types.Model)
			if ok {
				schema = string(m.Content)
			} else {
				return xerrors.Errorf("invalid schema: %v", m)
			}
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
		dataId, ok := value.(string)
		if ok {
			value, err = mm.CacheSvc.Get(account, dataId)
			if err != nil {
				log.Warn(err.Error())
			}

			if value == nil {
				return nil
			}
		}

		model, ok := value.(*types.Model)
		if ok {
			if len(model.Content) == 0 && len(model.Shards) > 0 {
				log.Warnf("large size content should go through P2P channel")
			}
			return model
		}
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
	mm.CacheSvc.Put(account, model.Alias+model.GroupId, model.DataId)

	// Reserved for open data model search feature...
	// for _, k := range model.Tags {
	// 	mm.CacheSvc.Put(account, k, model.DataId)
	// }
}
