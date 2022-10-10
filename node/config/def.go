package config

import (
	"bytes"

	"github.com/BurntSushi/toml"
	"golang.org/x/xerrors"
)

// storage node config
type StorageNode struct {
	Chain Chain
}

type Chain struct {
	Remote     string
	WsEndpoint string
}

func DefaultNode() *StorageNode {
	return &StorageNode{
		Chain: Chain{
			Remote:     "http://localhost:26657",
			WsEndpoint: "/websocket",
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
