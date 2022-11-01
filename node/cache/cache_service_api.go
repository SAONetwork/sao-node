package cache

import (
	"sync"

	logging "github.com/ipfs/go-log/v2"
)

type CacheSvcApi interface {
	CreateCache(name string, capacity int) error
	Get(name string, key string) (interface{}, error)
	Put(name string, key string, value interface{})
	Evict(name string, key string)
	GetSize(name string) int
	ReSize(name string, capacity int) error
}

var (
	once sync.Once
	log  = logging.Logger("cache")
)
