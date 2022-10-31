package cache

import (
	hamt "github.com/raviqqe/hamt"

	"golang.org/x/xerrors"
)

type LruCacheSvc struct {
	Caches map[string]*LruCache
}

var (
	lruCacheSvc *LruCacheSvc
)

func NewLruCacheSvc() *LruCacheSvc {
	once.Do(func() {
		lruCacheSvc = &LruCacheSvc{
			Caches: make(map[string]*LruCache),
		}
	})
	return lruCacheSvc
}

func (svc *LruCacheSvc) CreateCache(name string, capacity int) error {
	if svc.Caches[name] != nil {
		return xerrors.Errorf("the cache [%s] is existing already", name)
	}

	svc.Caches[name] = CreateLruCache(capacity)

	return nil
}

func (svc *LruCacheSvc) Get(name string, key string) (interface{}, error) {
	cache := svc.Caches[name]
	if cache == nil {
		return nil, xerrors.Errorf("the cache [%s] not found", name)
	}

	return cache.get(hamt.Entry(entryString(key))), nil
}

func (svc *LruCacheSvc) Put(name string, key string, value interface{}) {
	cache := svc.Caches[name]
	if cache == nil {
		log.Errorf("the cache [%s] not found", name)
	}

	cache.put(hamt.Entry(entryString(key)), value)
}

func (svc *LruCacheSvc) Evict(name string, key string) {
	cache := svc.Caches[name]
	if cache == nil {
		log.Errorf("the cache [%s] not found", name)
	}

	cache.evict(hamt.Entry(entryString(key)))
}

func (svc *LruCacheSvc) GetCapacity(name string) int {
	cache := svc.Caches[name]
	if cache == nil {
		log.Errorf("the cache [%s] not found", name)

		return 0
	}
	return cache.Capacity
}

func (svc *LruCacheSvc) GetSize(name string) int {
	cache := svc.Caches[name]
	if cache == nil {
		log.Errorf("the cache [%s] not found", name)

		return 0
	}
	return cache.Size
}

func (svc *LruCacheSvc) ReSize(name string, capacity int) error {
	cache := svc.Caches[name]
	if cache == nil {
		return xerrors.Errorf("the cache [%s] not found", name)
	}

	if capacity == -1 || cache.Capacity <= capacity {
		cache.Capacity = capacity
	} else {
		for {
			if cache.Map.Size() <= cache.Capacity {
				break
			}
			oldKey := cache.removeNode(cache.head)
			cache.Map.Delete(oldKey)
		}
	}

	return nil
}
