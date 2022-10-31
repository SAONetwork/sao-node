package cache

import (
	hamt "github.com/raviqqe/hamt"
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
		if l.Capacity > 0 && l.Map.Size() >= l.Capacity {
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
		l.Map = l.Map.Insert(key, node)
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

func CreateLruCache(capacity int) *LruCache {
	lruCache := LruCache{Capacity: capacity}
	lruCache.Map = hamt.NewMap()
	lruCache.Size = lruCache.Map.Size()
	return &lruCache
}
