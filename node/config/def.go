package config

import (
	"time"
)

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
		SaoHttpFileServer: SaoHttpFileServer{
			Enable:                  true,
			HttpFileServerAddress:   "localhost:5152",
			HttpFileServerPath:      "~/.sao-node/http-files",
			EnableHttpFileServerLog: false,
			TokenPeriod:             24 * time.Hour,
		},
		Storage: Storage{
			AcceptOrder: true,
			Ipfs:        []Ipfs{{Conn: "Conn"}},
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
	}
}
