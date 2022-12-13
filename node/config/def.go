package config

import (
	"time"
)

type Common struct {
	Chain  Chain
	Libp2p Libp2p
}

// gateway node config
type Node struct {
	Common
	Cache             Cache
	Transport         Transport
	Api               API
	Module            Module
	SaoHttpFileServer SaoHttpFileServer
	Storage           Storage
	SaoIpfs           SaoIpfs
}

type SaoHttpFileServer struct {
	Enable                  bool
	HttpFileServerAddress   string
	HttpFileServerPath      string
	EnableHttpFileServerLog bool
	TokenPeriod             time.Duration
}

type SaoIpfs struct {
	Enable bool
	Repo   string
}

type Storage struct {
	AcceptOrder bool
	Ipfs        []Ipfs
}

type Ipfs struct {
	Conn string
}

type Module struct {
	GatewayEnable bool
	StorageEnable bool
}

type API struct {
	ListenAddress    string
	Timeout          time.Duration
	EnablePermission bool
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
	MemcachedConn string
}

type Transport struct {
	TransportListenAddress []string
	StagingPath            string
	StagingSapceSize       int64
}

func DefaultSaoNode() *Node {
	return &Node{
		Common: defCommon(),
		Api: API{
			ListenAddress:    "/ip4/127.0.0.1/tcp/5151/http",
			Timeout:          30 * time.Second,
			EnablePermission: false,
		},
		Cache: Cache{
			EnableCache:   true,
			CacheCapacity: 1000,
			ContentLimit:  2 * 1024 * 1024,
		},
		Transport: Transport{
			TransportListenAddress: []string{
				"/ip4/0.0.0.0/udp/5154",
			},
			StagingPath:      "~/.sao-node/staging",
			StagingSapceSize: 32 * 1024 * 1024 * 1024,
		},
		Module: Module{
			GatewayEnable: true,
			StorageEnable: true,
		},
		SaoHttpFileServer: SaoHttpFileServer{
			Enable:                  true,
			HttpFileServerAddress:   "localhost:5152",
			HttpFileServerPath:      "~/.sao-node/http-files",
			EnableHttpFileServerLog: false,
			TokenPeriod:             24 * time.Hour,
		},
		Storage: Storage{
			AcceptOrder: true,
			Ipfs:        []Ipfs{},
		},
		SaoIpfs: SaoIpfs{
			Enable: true,
			Repo:   "~/.sao-node/ipfs",
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
				"/ip4/0.0.0.0/tcp/5153",
			},
		},
	}
}
