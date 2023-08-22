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
			EnableHttpFileServerLog: false,
			TokenPeriod:             24 * time.Hour,
		},
		Storage: Storage{
			AcceptOrder: true,
			Ipfs:        []Ipfs{},
		},
		SaoIpfs: SaoIpfs{
			Enable: true,
		},
		Indexer: Indexer{
			DbPath:        "~/.sao-node/datastore",
			ListenAddress: "localhost:5155",
		},
	}
}

func defCommon() Common {
	return Common{
		Chain: Chain{
			Remote:     "http://localhost:26657",
			WsEndpoint: "/websocket",
			TxPoolSize: 0,
		},
		Libp2p: Libp2p{
			ListenAddress: []string{
				"/ip4/0.0.0.0/tcp/5153",
			},
			AnnounceAddresses: []string{},
			PublicAddress:     "",
			IntranetIpEnable:  true,
			ExternalIpEnable:  true,
		},
		Transport: Transport{
			TransportListenAddress: []string{
				"/ip4/0.0.0.0/udp/5154",
			},
			StagingSapceSize: 32 * 1024 * 1024 * 1024,
		},
		Module: Module{
			GatewayEnable: false,
			StorageEnable: true,
			IndexerEnable: false,
		},
	}
}
