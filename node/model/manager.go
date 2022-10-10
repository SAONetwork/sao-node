package model

import (
	"fmt"
	"sao-storage-node/node/cache"
	"sao-storage-node/node/config"
	"sao-storage-node/node/model/json_patch"
	"sao-storage-node/node/model/schema/validator"
	"strings"
	"sync"

	cid "github.com/ipfs/go-cid"
	jsoniter "github.com/json-iterator/go"
	"github.com/multiformats/go-multihash"
	"golang.org/x/xerrors"
)

type Model struct {
	ResourceId string
	Alias      string
	Schema     []byte
	Content    []byte
	OrderId    string
	Cid        cid.Cid
}

type ModelManager struct {
	CacheCfg     *config.Cache
	CacheSvc     *cache.CacheSvc
	JsonpatchSvc *json_patch.JsonpatchSvc
	//CommitSvc *commit.CommitSvc
}

var (
	modelManager *ModelManager
	once         sync.Once
)

func NewModelManager(cacheCfg *config.Cache) *ModelManager {
	once.Do(func() {
		modelManager = &ModelManager{
			CacheCfg:     cacheCfg,
			CacheSvc:     cache.NewCacheSvc(),
			JsonpatchSvc: json_patch.NewJsonpatchSvc(),
			// CommitSvc: commit.NewCommitSvc(),
		}
	})

	return modelManager
}

func (m *ModelManager) Load(account string, alias string) (*Model, error) {
	model, err := m.CacheSvc.Get(account, alias)
	if model != nil {
		return model.(*Model), nil
	}

	if strings.Contains(err.Error(), fmt.Sprintf("the cache [%s] not found", account)) {
		err = m.CacheSvc.CreateCache(account, m.CacheCfg.CacheCapacity)
		if err != nil {
			return nil, xerrors.Errorf(err.Error())
		}
	}

	mm := model.(*Model)

	m.cacheModel(account, alias, mm)

	return mm, nil
}

func (m *ModelManager) Create(account string, alias string, content string, rule string) (*Model, error) {
	oldModel, err := m.CacheSvc.Get(account, alias)
	if oldModel != nil {
		return nil, xerrors.Errorf("the model [%s] is exsiting already", alias)
	}
	if strings.Contains(err.Error(), fmt.Sprintf("the cache [%s] not found", account)) {
		err = m.CacheSvc.CreateCache(account, m.CacheCfg.CacheCapacity)
		if err != nil {
			return nil, xerrors.Errorf(err.Error())
		}
	}

	err = m.validateModel(account, alias, []byte(content), rule)
	if err != nil {
		return nil, xerrors.Errorf(err.Error())
	}

	// model, err = m.CommitSvc.Commit(account, content)
	// if err != nil {
	// 	return nil, xerrors.Errorf(err.Error())
	// }

	model := &Model{
		ResourceId: GenerateResourceId(),
		Alias:      alias,
		Content:    []byte(content),
		Cid:        cid.NewCidV1(cid.Raw, multihash.Multihash(alias)),
	}

	m.cacheModel(account, alias, model)

	return model, nil
}

func (m *ModelManager) Update(account string, alias string, patch string, rule string) (*Model, error) {
	orgModel, err := m.CacheSvc.Get(account, alias)
	if err != nil || orgModel == nil {
		return nil, xerrors.Errorf(err.Error())
	}

	contentBytes, err := jsoniter.Marshal(orgModel)
	if err != nil {
		return nil, xerrors.Errorf(err.Error())
	}

	newContentBytes, err := m.JsonpatchSvc.ApplyPatch(contentBytes, []byte(patch))
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
		ResourceId: GenerateResourceId(),
		Alias:      alias,
		Content:    newContentBytes,
		Cid:        cid.NewCidV1(cid.Raw, multihash.Multihash(alias)),
	}

	m.cacheModel(account, alias, model)

	return model, nil
}

func (m *ModelManager) validateModel(account string, alias string, contentBytes []byte, rule string) error {
	schema := jsoniter.Get(contentBytes, "@context").ToString()
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
	} else {
		m.CacheSvc.Put(account, alias, model)
	}
	m.CacheSvc.Put(account, model.ResourceId, alias)
	m.CacheSvc.Put(account, model.OrderId, alias)
}
