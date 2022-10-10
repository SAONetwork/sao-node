package config

import (
	"bytes"

	"github.com/BurntSushi/toml"
	"golang.org/x/xerrors"
)

// storage node config
type StorageNode struct {
	Chain Chain
	Cache Cache
}

type Chain struct {
	Remote     string
	WsEndpoint string
}

type Cache struct {
	CacheCapacity int
	ContentLimit  int
}

func DefaultNode() *StorageNode {
	return &StorageNode{
		Chain: Chain{
			Remote:     "http://localhost:26657",
			WsEndpoint: "/websocket",
		},
		Cache: Cache{
			CacheCapacity: 1000,
			ContentLimit:  2097152,
		},
	}
}

func NodeBytes(cfg interface{}) ([]byte, error) {
	buf := new(bytes.Buffer)
	e := toml.NewEncoder(buf)
	if err := e.Encode(cfg); err != nil {
		return nil, xerrors.Errorf("encoding node config: %w", err)
	}

	return []byte(buf.String()), nil
}
