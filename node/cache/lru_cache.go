package cache

import (
	hamt "github.com/raviqqe/hamt"
)

type entryString string

// FNV Hash prime
const primeRK = 16777619

func (s entryString) Hash() uint32 {
	return GetHash32(s)
}

func (i entryString) Equal(e hamt.Entry) bool {
	j, ok := e.(entryString)

	if !ok {
		return false
	}

	return i == j
}

func GetHash32(dataStr entryString) uint32 {
	data := []byte(dataStr)

	hash := uint32(0)
	for i := 0; i < len(data); i++ {
		//hash = 0 * 16777619 + sep[i]
		hash = hash*primeRK + uint32(data[i])
	}
	var pow, sq uint32 = 1, primeRK
	for i := len(data); i > 0; i >>= 1 {
		if i&1 != 0 {
			pow *= sq
		}
		sq *= sq
	}
	return hash
}

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

func (l *LruCache) get(key string) interface{} {
	value := l.Map.Find(hamt.Entry(entryString(key)))
	if value != nil {
		node, ok := value.(*Node)
		if ok {
			l.refreshNode(node)
			return node.Value
		}
	}

	return nil
}

func (l *LruCache) put(keyStr string, value interface{}) {
	key := hamt.Entry(entryString(keyStr))
	oldValue := l.Map.Find(key)
	if oldValue == nil {
		node := Node{Key: key, Value: value}
		if l.Capacity > 0 && l.Map.Size() >= l.Capacity {
			oldKey := l.removeNode(l.head)
			l.Map = l.Map.Delete(oldKey).Insert(key, &node)
		} else {
			l.Map = l.Map.Insert(key, &node)
		}
		l.addNode(&node)
	} else {
		node, ok := oldValue.(*Node)
		if ok {
			node.Value = value
			l.refreshNode(node)
			l.Map = l.Map.Insert(key, node)
		} else {
			return
		}
	}
	l.Size = l.Map.Size()
}

func (l *LruCache) evict(key string) {
	value := l.Map.Find(hamt.Entry(entryString(key)))
	if value != nil {
		node, ok := value.(*Node)
		if ok {
			oldKey := l.removeNode(node)
			l.Map = l.Map.Delete(oldKey)
			l.Size = l.Map.Size()
		}
	}
}

func CreateLruCache(capacity int) *LruCache {
	lruCache := LruCache{Capacity: capacity}
	lruCache.Map = hamt.NewMap()
	lruCache.Size = lruCache.Map.Size()
	return &lruCache
}
