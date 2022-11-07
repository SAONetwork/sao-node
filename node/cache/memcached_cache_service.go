package cache

import (
	"encoding/json"

	"github.com/bradfitz/gomemcache/memcache"
	"golang.org/x/xerrors"
)

type MemcachedCacheSvc struct {
	Client *memcache.Client
}

var (
	memcacheCacheSvc *MemcachedCacheSvc
)

func NewMemcachedCacheSvc(conn string) *MemcachedCacheSvc {
	once.Do(func() {
		log.Infof("octopus: init memcache client: %v ******", conn)

		cli := memcache.New(conn)

		if cli != nil {
			memcacheCacheSvc = &MemcachedCacheSvc{
				Client: cli,
			}
		}
	})
	return memcacheCacheSvc
}

func (svc *MemcachedCacheSvc) CreateCache(name string, capacity int) error {
	return nil
}

func (svc *MemcachedCacheSvc) Get(name string, key string) (interface{}, error) {
	item, err := svc.Client.Get(name + "_" + key)
	if err != nil {
		return nil, err
	}

	if item.Value != nil {
		var res interface{}
		err := json.Unmarshal(item.Value, &res)
		if err != nil {
			log.Error(err.Error())
			return res, nil
		}
	}

	return nil, xerrors.Errorf("not found")
}

func (svc *MemcachedCacheSvc) Put(name string, key string, value interface{}) {
	bytes, err := json.Marshal(value)
	if err != nil {
		log.Error(err.Error())
		return
	}

	err = svc.Client.Set(&memcache.Item{
		Key:   name + "_" + key,
		Value: bytes,
		Flags: 0,
	})
	if err != nil {
		log.Error(err.Error())
	}
}

func (svc *MemcachedCacheSvc) Evict(name string, key string) {
	err := svc.Client.Delete(name + "_" + key)
	if err != nil {
		log.Error(err.Error())
	}
}

func (svc *MemcachedCacheSvc) GetCapacity(name string) int {
	log.Warn("depends on memcache capacity")

	return -1
}

func (svc *MemcachedCacheSvc) GetSize(name string) int {
	log.Warn("depends on memcache capacity")

	return -1
}

func (svc *MemcachedCacheSvc) ReSize(name string, capacity int) error {
	log.Warn("unsupport operation, depends on memcache capacity")

	return nil
}
