package config

import "time"

type Common struct {
	Chain     Chain
	Libp2p    Libp2p
	Module    Module
	Transport Transport
}

type Node struct {
	Common

	Cache             Cache
	SaoHttpFileServer SaoHttpFileServer
	Api               API

	Storage Storage
	SaoIpfs SaoIpfs
	Indexer Indexer
}

type SaoHttpFileServer struct {
	Enable                  bool
	HttpFileServerAddress   string
	HttpFileServerPath      string
	EnableHttpFileServerLog bool
	TokenPeriod             time.Duration
}

// SaoIpfs contains configs for inprocess ipfs
type SaoIpfs struct {
	// Enable in process ipfs instance
	Enable bool
	// ipfs repo path
	Repo string
}

// Storage contains configs for backend storages
type Storage struct {

	// if this node is open to accept order shards
	AcceptOrder bool
	Ipfs        []Ipfs
}

// Ipfs contains configs for backend ipfs
type Ipfs struct {

	// ipfs connection string
	Conn string
}

// Indexer contains configs for indexing and graphsql service
type Indexer struct {
	// indexer db path
	DbPath string

	// Binding address for the graphsql service
	ListenAddress string
}

// Module contains configs for Submodules
type Module struct {

	// Enable gateway module
	GatewayEnable bool

	// Enable storage module
	StorageEnable bool

	// Enable indexer module
	IndexerEnable bool
}

// API contains configs for API endpoint
type API struct {

	// Binding address for the Sao Node API
	ListenAddress string

	Timeout time.Duration

	EnablePermission bool
}

// Chain contains configs for sao chain information
type Chain struct {

	// remote connection string
	Remote string

	// websocket endpoint
	WsEndpoint string
}

// Libp2p contains configs for libp2p
type Libp2p struct {
	// Binding address for the libp2p host - 0 means random port.
	// Format: multiaddress; see https://multiformats.io/multiaddr/
	ListenAddress     []string
	AnnounceAddresses []string
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
