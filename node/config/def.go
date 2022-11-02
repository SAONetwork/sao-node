package config

import (
	"bytes"

	"time"

	"github.com/BurntSushi/toml"
	"golang.org/x/xerrors"
)

type Common struct {
	Chain  Chain
	Libp2p Libp2p
}

// gateway node config
type Node struct {
	Common
	Cache     Cache
	Transport Transport
	Api       API
	Module    Module
	Storage   Storage
	SaoIpfs   SaoIpfs
}

type SaoIpfs struct {
	Enable bool
	Repo   string
}

type Storage struct {
	Ipfs []Ipfs
}

type Ipfs struct {
	Conn string
}

type Module struct {
	GatewayEnable bool
	StorageEnable bool
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
	EnableCache   bool
	CacheCapacity int
	ContentLimit  int
	RedisConn     string
	RedisPassword string
	RedisPoolSize int
}

type Transport struct {
	TransportListenAddress []string
	StagingPath            string
	StagingSapceSize       int64
}

func DefaultGatewayNode() *Node {
	return &Node{
		Common: defCommon(),
		Api: API{
			ListenAddress: "/ip4/127.0.0.1/tcp/8888/http",
			Timeout:       30 * time.Second,
		},
		Cache: Cache{
			EnableCache:   true,
			CacheCapacity: 1000,
			ContentLimit:  2 * 1024 * 1024,
		},
		Transport: Transport{
			TransportListenAddress: []string{
				"/ip4/0.0.0.0/udp/26660",
			},
			StagingPath:      "~/.sao_staging",
			StagingSapceSize: 32 * 1024 * 1024 * 1024,
		},
		Module: Module{
			GatewayEnable: true,
			StorageEnable: true,
		},
		Storage: Storage{
			Ipfs: []Ipfs{},
		},
		SaoIpfs: SaoIpfs{
			Enable: true,
			Repo:   "~/.sao_ipfs",
		},
	}
}

func defCommon() Common {
	return Common{
		Chain: Chain{
			Remote:        "http://localhost:26657",
			WsEndpoint:    "/websocket",
			AddressPrefix: "cosmos",
		},
		Libp2p: Libp2p{
			ListenAddress: []string{
				"/ip4/0.0.0.0/tcp/26659",
			},
		},
	}
}

func NodeBytes(cfg interface{}) ([]byte, error) {
	buf := new(bytes.Buffer)
	e := toml.NewEncoder(buf)
	if err := e.Encode(cfg); err != nil {
		return nil, xerrors.Errorf("encoding node config: %w", err)
	}

	return buf.Bytes(), nil
}
