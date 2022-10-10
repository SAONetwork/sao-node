package model

import (
	"fmt"
	"sao-storage-node/node/cache"
	"sao-storage-node/node/config"
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
	CacheCfg *config.Cache
	CacheSvc *cache.CacheSvc
	//CommitSvc *commit.CommitSvc
}

var (
	modelManager *ModelManager
	once         sync.Once
)

func NewModelManager(cacheCfg *config.Cache) *ModelManager {
	once.Do(func() {
		modelManager = &ModelManager{
			CacheCfg: cacheCfg,
			CacheSvc: cache.NewCacheSvc(),
			// CommitSvc: commit.NewCommitSvc(),
		}
	})

	return modelManager
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

	contentBytes := []byte(content)
	schema := jsoniter.Get(contentBytes, "@context").ToString()
	validator, err := validator.NewDataModelValidator(alias, schema, rule)
	if err != nil {
		return nil, xerrors.Errorf(err.Error())
	}
	err = validator.Validate(jsoniter.Get(contentBytes))
	if err != nil {
		return nil, xerrors.Errorf(err.Error())
	}

	var model *Model
	// model, err = m.CommitSvc(account, content)
	// if err != nil {
	// 	return nil, xerrors.Errorf(err.Error())
	// }

	model = &Model{
		ResourceId: GenerateResourceId(),
		Alias:      alias,
		Content:    []byte(content),
		Cid:        cid.NewCidV1(cid.Raw, multihash.Multihash(alias)),
	}

	if len(contentBytes) > m.CacheCfg.ContentLimit {
		m.CacheSvc.Put(account, alias, model.Cid.String())
	} else {
		m.CacheSvc.Put(account, alias, model)
	}
	m.CacheSvc.Put(account, model.ResourceId, alias)
	m.CacheSvc.Put(account, model.OrderId, alias)

	return model, nil
}
