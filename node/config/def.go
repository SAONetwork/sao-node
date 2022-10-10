package config

import (
	"bytes"

	"time"

	"github.com/BurntSushi/toml"
	"golang.org/x/xerrors"
)

// storage node config
type StorageNode struct {
	Cache  Cache
	Chain  Chain
	Libp2p Libp2p
	Api    API
}

type API struct {
	ListenAddress string
	Timeout       time.Duration
}

type Chain struct {
	Remote        string
	WsEndpoint    string
	AddressPrefix string
}

type Libp2p struct {
	// Binding address for the libp2p host - 0 means random port.
	// Format: multiaddress; see https://multiformats.io/multiaddr/
	ListenAddress []string
}

type Cache struct {
	CacheCapacity int
	ContentLimit  int
}

func DefaultNode() *StorageNode {
	return &StorageNode{
		Chain: Chain{
			Remote:        "http://localhost:26657",
			WsEndpoint:    "/websocket",
			AddressPrefix: "cosmos",
		},
		Libp2p: Libp2p{
			ListenAddress: []string{
				"/ip4/0.0.0.0/tcp/0",
				"/ip6/::/tcp/0",
			},
		},
		Api: API{
			ListenAddress: "/ip4/127.0.0.1/tcp/8888/http",
			Timeout:       30 * time.Second,
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
