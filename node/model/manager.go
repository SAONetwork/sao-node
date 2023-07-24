package model

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/SaoNetwork/sao-node/node/cache"
	"github.com/SaoNetwork/sao-node/node/config"
	"github.com/SaoNetwork/sao-node/node/gateway"
	"github.com/SaoNetwork/sao-node/node/model/schema/validator"
	"github.com/SaoNetwork/sao-node/types"
	"github.com/SaoNetwork/sao-node/utils"

	saotypes "github.com/SaoNetwork/sao/x/sao/types"
	logging "github.com/ipfs/go-log/v2"
	jsoniter "github.com/json-iterator/go"
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
		if cacheCfg.RedisConn == "" && cacheCfg.MemcachedConn == "" {
			cacheSvc = cache.NewLruCacheSvc()
		} else if cacheCfg.RedisConn != "" {
			cacheSvc = cache.NewRedisCacheSvc(cacheCfg.RedisConn, cacheCfg.RedisPassword, cacheCfg.RedisPoolSize)
		} else if cacheCfg.MemcachedConn != "" {
			cacheSvc = cache.NewMemcachedCacheSvc(cacheCfg.MemcachedConn)
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

func (mm *ModelManager) Load(ctx context.Context, req *types.MetadataProposal) (*types.Model, error) {
	log.Info("KeyWord:", req.Proposal.Keyword)

	var queryCommitId string
	var model *types.Model
	if req.Proposal.CommitId != "" {
		queryCommitId = req.Proposal.CommitId

		log.Infof("load model, account: %s, key: %s", req.Proposal.Owner, req.Proposal.Keyword+queryCommitId)
		model = mm.loadModel(req.Proposal.Owner, req.Proposal.Keyword+queryCommitId)
		if model != nil {
			log.Infof("Cache hit, model[%s, %s]-%s", model.DataId, model.CommitId, model.Alias)
			if (req.Proposal.CommitId == "" || model.CommitId == req.Proposal.CommitId) && len(model.Content) > 0 {
				log.Debug("model", model)
				// found latest data model in local cache already
				log.Debugf("load the model[%s]-%s from cache", model.DataId, model.Alias)
				log.Debug("model: ", string(model.Content))
				return model, nil
			} else {
				log.Infof("not model %s:%s found in the cache, fetch it from the network", req.Proposal.Keyword, req.Proposal.CommitId)
				log.Infof("local version model is %s:%s.", model.DataId, model.CommitId)
			}
		}
	}

	meta, err := mm.GatewaySvc.QueryMeta(ctx, req, 0)
	if err != nil {
		return nil, err
	}

	model = mm.loadModel(meta.Owner, req.Proposal.Keyword+meta.CommitId)
	if model != nil {
		log.Infof("Cache hit, model[%s, %s]-%s", model.DataId, model.CommitId, model.Alias)
		if (req.Proposal.CommitId == "" || model.CommitId == req.Proposal.CommitId) && len(model.Content) > 0 {
			log.Debug("model", model)
			// found latest data model in local cache already
			log.Debugf("load the model[%s]-%s from cache", model.DataId, model.Alias)
			log.Debug("model: ", string(model.Content))
			return model, nil
		} else {
			log.Infof("not model %s:%s found in the cache, fetch it from the network", req.Proposal.Keyword, req.Proposal.CommitId)
			log.Infof("local version model is %s:%s.", model.DataId, model.CommitId)
		}
	}

	version := req.Proposal.Version
	if req.Proposal.Version != "" {
		match, err := regexp.Match(`^v\d+$`, []byte(req.Proposal.Version))
		if err != nil || !match {
			return nil, types.Wrapf(types.ErrInvalidVersion, "invalid Version: %s", req.Proposal.Version)
		}

		index, err := strconv.Atoi(strings.ReplaceAll(req.Proposal.Version, "v", ""))
		if err != nil {
			return nil, types.Wrap(types.ErrInvalidVersion, err)
		}

		if len(meta.Commits) > index {
			commit := meta.Commits[index]
			commitInfo, err := types.ParseMetaCommit(commit)
			if err != nil {
				return nil, types.Wrapf(types.ErrInvalidCommitInfo, "invalid commit information: %s", commit)
			}
			meta, err = mm.GatewaySvc.QueryMeta(ctx, req, int64(commitInfo.Height))
			if err != nil {
				return nil, err
			}
		} else {
			return nil, types.Wrapf(types.ErrInvalidVersion, "invalid Version: %s", req.Proposal.Version)
		}
	} else {
		version = fmt.Sprintf("v%d", len(meta.Commits)-1)
	}

	if req.Proposal.CommitId != "" {
		isFound := false
		for i, commit := range meta.Commits {
			commitInfo, err := types.ParseMetaCommit(commit)
			if err != nil {
				return nil, types.Wrapf(types.ErrInvalidCommitInfo, "invalid commit information: %s", commit)
			}

			if commitInfo.CommitId == req.Proposal.CommitId {
				meta, err = mm.GatewaySvc.QueryMeta(ctx, req, int64(commitInfo.Height))
				if err != nil {
					return nil, err
				}

				version = fmt.Sprintf("v%d", i)
				isFound = true
				break
			}
		}

		if !isFound {
			return nil, types.Wrapf(types.ErrInvalidCommitInfo, "invalid CommitId: %s", req.Proposal.CommitId)
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
	} else {
		model.OrderId = meta.OrderId
		model.Cid = meta.Cid
		model.Shards = meta.Shards
		model.CommitId = meta.CommitId
		model.Commits = meta.Commits
		model.ExtendInfo = meta.ExtendInfo
	}

	result, err := mm.GatewaySvc.FetchContent(ctx, req, meta)
	if err != nil {
		return nil, err
	}
	model.Cid = result.Cid
	model.Content = result.Content
	model.Version = version

	mm.cacheModel(req.Proposal.Owner, model)

	return model, nil
}

func (mm *ModelManager) Create(ctx context.Context, req *types.MetadataProposal, clientProposal *types.OrderStoreProposal, orderId uint64, content []byte) (*types.Model, error) {
	orderProposal := clientProposal.Proposal
	if orderProposal.Alias == "" {
		orderProposal.Alias = orderProposal.Cid
	}

	oldModel := mm.loadModel(orderProposal.Owner, orderProposal.DataId+orderProposal.DataId)
	if oldModel != nil {
		return nil, types.Wrapf(types.ErrInvalidDataId, "the model is exsiting already, alias: %s, dataId: %s", oldModel.Alias, oldModel.DataId)
	}

	oldModel = mm.loadModel(orderProposal.Owner, orderProposal.Alias+orderProposal.DataId)
	if oldModel != nil {
		return nil, types.Wrapf(types.ErrInvalidDataId, "the model is exsiting already, alias: %s, dataId: %s", oldModel.Alias, oldModel.DataId)
	}

	meta, err := mm.GatewaySvc.QueryMeta(ctx, req, 0)
	log.Infof("orderProposal: %#v", orderProposal)
	log.Infof("meta: %#v", meta)
	if err == nil && meta != nil {
		return nil, types.Wrapf(types.ErrConflictId, "the model is exsiting already, alias: %s, dataId: %s", meta.Alias, meta.DataId)
	}

	if orderProposal.Size_ == 0 || len(content) == 0 {
		return nil, types.Wrapf(types.ErrInvalidContent, "the content is empty")
	}

	err = mm.validateModel(ctx, orderProposal.Owner, orderProposal.Alias, content, orderProposal.Rule)
	if err != nil {
		return nil, err
	}

	// Commit
	result, err := mm.GatewaySvc.CommitModel(ctx, clientProposal, orderId, content)
	if err != nil {
		return nil, err
	}

	commit := bytes.NewBufferString(orderProposal.CommitId)
	commit.WriteByte(26)
	commit.WriteString(fmt.Sprintf("%d", result.Height))

	model := &types.Model{
		DataId:     result.DataId,
		Alias:      orderProposal.Alias,
		GroupId:    orderProposal.GroupId,
		OrderId:    result.OrderId,
		Owner:      orderProposal.Owner,
		Tags:       orderProposal.Tags,
		Cid:        result.Cid,
		Shards:     result.Shards,
		CommitId:   orderProposal.CommitId,
		Commits:    append(make([]string, 0), commit.String()),
		Version:    "v0",
		Content:    content,
		ExtendInfo: orderProposal.ExtendInfo,
	}

	mm.cacheModel(orderProposal.Owner, model)
	log.Infof("create model[%s, %s]-%s cached", model.DataId, model.CommitId, model.Alias)

	return model, nil
}

func (mm *ModelManager) Update(ctx context.Context, req *types.MetadataProposal, clientProposal *types.OrderStoreProposal, orderId uint64, patch []byte) (*types.Model, error) {
	commitIds := strings.Split(clientProposal.Proposal.CommitId, "|")
	if len(commitIds) != 2 {
		return nil, types.Wrapf(types.ErrInvalidCommitInfo, "invalid commitId:%s", clientProposal.Proposal.CommitId)
	}
	lastCommitId := commitIds[0]

	var isFetch = true
	meta, err := mm.GatewaySvc.QueryMeta(ctx, req, 0)
	if err != nil {
		return nil, err
	}

	orgModel := mm.loadModel(meta.Owner, req.Proposal.Keyword+lastCommitId)
	if orgModel != nil {
		if lastCommitId == orgModel.CommitId && len(orgModel.Content) > 0 {
			if meta.CommitId == orgModel.CommitId {
				// found latest data model in local cache already
				log.Debugf("load the model[%s]-%s from cache", orgModel.DataId, orgModel.Alias)
				log.Debug("model: ", string(orgModel.Content))
				isFetch = false
			} else {
				log.Debugf("the local cached model is out of date")
			}
		} else {
			log.Debugf("not model %s:%s found in the cache, fetch it from the network", orgModel.DataId, lastCommitId)
			log.Debugf("local version model is %s:%s.", orgModel.DataId, orgModel.CommitId)
		}
	}

	if isFetch {
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

		result, err := mm.GatewaySvc.FetchContent(ctx, req, meta)
		if err != nil {
			return nil, err
		}
		log.Info("result: ", result)
		log.Info("orgModel: ", orgModel)
		orgModel.Content = result.Content
	}

	log.Debug("orgModel: ", string(orgModel.Content))
	log.Debug("patch: ", string(patch))
	newContent, err := utils.ApplyPatch(orgModel.Content, []byte(patch))
	if err != nil {
		return nil, err
	}
	log.Debug("newContent: ", string(newContent))
	if bytes.Equal(orgModel.Content, newContent) {
		return nil, types.Wrapf(types.ErrInvalidContent, "no content updated.")
	}

	if len(newContent) != int(clientProposal.Proposal.Size_) {
		return nil, types.Wrapf(types.ErrInvalidContent, "given size(%d) doesn't match target content size(%d)", int(clientProposal.Proposal.Size_), len(newContent))
	}

	newContentCid, err := utils.CalculateCid(newContent)
	if err != nil {
		return nil, err
	}
	if newContentCid.String() != clientProposal.Proposal.Cid {
		return nil, types.Wrapf(types.ErrInvalidCid, "cid mismatch, expected %s, but got %s", clientProposal.Proposal.Cid, newContentCid)
	}

	err = mm.validateModel(ctx, clientProposal.Proposal.Owner, clientProposal.Proposal.Alias, newContent, clientProposal.Proposal.Rule)
	if err != nil {
		return nil, err
	}

	// Commit
	result, err := mm.GatewaySvc.CommitModel(ctx, clientProposal, orderId, newContent)
	if err != nil {
		return nil, err
	}
	log.Debug("CommitedModel!!!")

	commit := bytes.NewBufferString(commitIds[1])
	commit.WriteByte(26)
	commit.WriteString(fmt.Sprintf("%d", result.Height))

	model := &types.Model{
		DataId:     orgModel.DataId,
		Alias:      orgModel.Alias,
		GroupId:    clientProposal.Proposal.GroupId,
		OrderId:    result.OrderId,
		Owner:      clientProposal.Proposal.Owner,
		Tags:       clientProposal.Proposal.Tags,
		Cid:        result.Cid,
		Shards:     result.Shards,
		CommitId:   commitIds[1],
		Commits:    append(orgModel.Commits, commit.String()),
		Version:    fmt.Sprintf("v%d", len(orgModel.Commits)),
		Content:    newContent,
		ExtendInfo: clientProposal.Proposal.ExtendInfo,
	}

	mm.cacheModel(clientProposal.Proposal.Owner, model)
	log.Infof("update model[%s, %s]-%s cached", model.DataId, model.CommitId, model.Alias)

	return model, nil
}

func (mm *ModelManager) Delete(ctx context.Context, req *types.OrderTerminateProposal, isPublish bool) (*types.Model, error) {
	if isPublish {
		err := mm.GatewaySvc.TerminateOrder(ctx, req)
		if err != nil {
			return nil, err
		}
	}

	model, _ := mm.CacheSvc.Get(req.Proposal.Owner, req.Proposal.DataId)
	if model != nil {
		m, ok := model.(*types.Model)
		if ok {
			mm.CacheSvc.Evict(req.Proposal.Owner, m.DataId)
			mm.CacheSvc.Evict(req.Proposal.Owner, m.Alias)

			return &types.Model{
				DataId: m.DataId,
				Alias:  m.Alias,
			}, nil
		}
	}

	return nil, nil
}

func (mm *ModelManager) ShowCommits(ctx context.Context, req *types.MetadataProposal) (*types.Model, error) {
	meta, err := mm.GatewaySvc.QueryMeta(ctx, req, 0)
	if err != nil {
		return nil, err
	}

	return &types.Model{
		DataId:  meta.DataId,
		Alias:   meta.Alias,
		Commits: meta.Commits,
	}, nil
}

func (mm *ModelManager) Renew(ctx context.Context, req *types.OrderRenewProposal, isPublish bool) (map[string]string, error) {
	if isPublish {
		results, err := mm.GatewaySvc.RenewOrder(ctx, req)
		if err != nil {
			return nil, err
		}
		return results, nil
	}

	return nil, nil
}

func (mm *ModelManager) UpdatePermission(ctx context.Context, req *types.PermissionProposal, isPublish bool) (*types.Model, error) {
	if isPublish {
		err := mm.GatewaySvc.UpdateModelPermission(ctx, req)
		if err != nil {
			return nil, err
		}
	}

	return &types.Model{
		DataId: req.Proposal.DataId,
	}, nil
}

func (mm *ModelManager) validateModel(ctx context.Context, account string, alias string, contentBytes []byte, rule string) error {
	schemaStr := jsoniter.Get(contentBytes, PROPERTY_CONTEXT).ToString()
	if schemaStr == "" {
		return nil
	}

	match, err := regexp.Match(`^\[.*\]$`, []byte(schemaStr))
	if err != nil {
		return types.Wrap(types.ErrInvalidSchema, err)
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
						return err
					}

					if model == nil {
						req := &types.MetadataProposal{
							Proposal: saotypes.QueryProposal{
								Owner:       "all",
								Keyword:     sch,
								KeywordType: 0,
							},
						}

						model, err = mm.Load(ctx, req)
						if err != nil {
							return err
						}
					}
					m, ok := model.(*types.Model)
					if ok {
						sch = string(m.Content)
					} else {
						return types.Wrapf(types.ErrInvalidSchema, "invalid schema: %v", m)
					}
				}

				validator, err := validator.NewDataModelValidator(alias, sch, rule)
				if err != nil {
					return err
				}
				err = validator.Validate(jsoniter.Get(contentBytes))
				if err != nil {
					return err
				}
			} else {
				return types.Wrapf(types.ErrInvalidSchema, "invalid schema: %v", schema)
			}
		}
	} else {
		iter := jsoniter.ParseString(jsoniter.ConfigDefault, schemaStr)
		dataId := iter.ReadString()
		var schema string
		if utils.IsDataId(dataId) {
			model, err := mm.CacheSvc.Get(account, dataId)
			if err != nil {
				return err
			}

			if model == nil {
				req := &types.MetadataProposal{
					Proposal: saotypes.QueryProposal{
						Owner:       "all",
						Keyword:     dataId,
						KeywordType: 0,
					},
				}

				model, err = mm.Load(ctx, req)
				if err != nil {
					return err
				}
			}

			m, ok := model.(*types.Model)
			if ok {
				schema = string(m.Content)
			} else {
				return types.Wrapf(types.ErrInvalidSchema, "invalid schema: %v", m)
			}
		} else {
			schema = iter.ReadObject()
		}

		validator, err := validator.NewDataModelValidator(alias, schema, rule)
		if err != nil {
			return err
		}
		err = validator.Validate(jsoniter.Get(contentBytes))
		if err != nil {
			return err
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
			buf, _ := json.Marshal(model)
			log.Debug("model: ", string(buf), " LOADED!!!")

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
	mm.CacheSvc.Put(account, model.DataId + model.CommitId, model)
	mm.CacheSvc.Put(account, model.Alias + model.CommitId, model)

	//mm.CacheSvc.Put("did:key:zQ3shiAGhyFEGS3WhS64PYU9GEBk1rtrzaApJbHFEmWQbp5Xg", model.DataId + model.CommitId, model)
	//mm.CacheSvc.Put("did:key:zQ3shiAGhyFEGS3WhS64PYU9GEBk1rtrzaApJbHFEmWQbp5Xg", model.Alias + model.CommitId, model)

	buf, _ := json.Marshal(model)
	log.Debug("model: ", string(buf), " CACHED!!!")
}
