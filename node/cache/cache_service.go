package cache

import (
	"sync"

	hamt "github.com/raviqqe/hamt"
	"golang.org/x/xerrors"
)

type (
	Node struct {
		Key   hamt.Entry
		Value interface{}
		pre   *Node
		next  *Node
	}

	LruCache struct {
		Capacity int
		Size     int
		head     *Node
		end      *Node

		Map hamt.Map
	}

	CacheSvc struct {
		Caches map[string]*LruCache
	}
)

var (
	cacheSvc *CacheSvc
	once     sync.Once
)

func (l *LruCache) addNode(node *Node) {
	if l.end != nil {
		l.end.next = node
		node.pre = l.end
		node.next = nil
	}
	l.end = node
	if l.head == nil {
		l.head = node
	}
}

func (l *LruCache) removeNode(node *Node) hamt.Entry {
	if node == l.end {
		l.end = l.end.pre
	} else if node == l.head {
		l.head = l.head.next
	} else {
		node.pre.next = node.next
		node.next.pre = node.pre
	}
	return node.Key
}

func (l *LruCache) refreshNode(node *Node) {
	if node == l.end {
		return
	}
	l.removeNode(node)
	l.addNode(node)
}

func CreateLruCache(capacity int) *LruCache {
	lruCache := LruCache{Capacity: capacity}
	lruCache.Map = hamt.NewMap()
	lruCache.Size = lruCache.Map.Size()
	return &lruCache
}

func (l *LruCache) get(key hamt.Entry) interface{} {
	value := l.Map.Find(key)
	if value != nil {
		node := value.(*Node)
		l.refreshNode(node)
		return node.Value
	} else {
		return nil
	}
}

func (l *LruCache) put(key hamt.Entry, value interface{}) {
	oldValue := l.Map.Find(key)
	if oldValue == nil {
		node := Node{Key: key, Value: value}
		if l.Map.Size() >= l.Capacity {
			oldKey := l.removeNode(l.head)
			l.Map = l.Map.Delete(oldKey).Insert(key, &node)
		} else {
			l.Map = l.Map.Insert(key, &node)
		}
		l.addNode(&node)
	} else {
		node := oldValue.(*Node)
		node.Value = value
		l.refreshNode(node)
		l.Map = l.Map.Insert(key, &node)
	}
	l.Size = l.Map.Size()
}

func (l *LruCache) evict(key hamt.Entry) {
	value := l.Map.Find(key)
	if value != nil {
		oldKey := l.removeNode(value.(*Node))
		l.Map = l.Map.Delete(oldKey)
		l.Size = l.Map.Size()
	}
}

func NewCacheSvc() *CacheSvc {
	once.Do(func() {
		cacheSvc = &CacheSvc{
			Caches: make(map[string]*LruCache),
		}
	})
	return cacheSvc
}

func (svc *CacheSvc) CreateCache(name string, capacity int) error {
	if svc.Caches[name] != nil {
		return xerrors.Errorf("the cache [%s] is existing already", name)
	}

	svc.Caches[name] = CreateLruCache(capacity)

	return nil
}

func (svc *CacheSvc) Get(name string, key string) (interface{}, error) {
	cache := svc.Caches[name]
	if cache == nil {
		return nil, xerrors.Errorf("the cache [%s] not found", name)
	}

	return cache.get(hamt.Entry(entryString(key))), nil
}

func (svc *CacheSvc) Put(name string, key string, value interface{}) error {
	cache := svc.Caches[name]
	if cache == nil {
		return xerrors.Errorf("the cache [%s] not found", name)
	}

	cache.put(hamt.Entry(entryString(key)), value)

	return nil
}

func (svc *CacheSvc) Evict(name string, key string) error {
	cache := svc.Caches[name]
	if cache == nil {
		return xerrors.Errorf("the cache [%s] not found", name)
	}

	cache.evict(hamt.Entry(entryString(key)))

	return nil
}

func (svc *CacheSvc) GetCapacity(name string) (int, error) {
	cache := svc.Caches[name]
	if cache == nil {
		return 0, xerrors.Errorf("the cache [%s] not found", name)
	}
	return cache.Capacity, nil
}

func (svc *CacheSvc) GetSize(name string) (int, error) {
	cache := svc.Caches[name]
	if cache == nil {
		return 0, xerrors.Errorf("the cache [%s] not found", name)
	}
	return cache.Size, nil
}

func (svc *CacheSvc) ReSize(name string, capacity int) error {
	cache := svc.Caches[name]
	if cache == nil {
		return xerrors.Errorf("the cache [%s] not found", name)
	}

	if cache.Capacity <= capacity {
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
