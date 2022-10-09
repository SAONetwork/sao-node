package config

import (
	"bytes"
	"github.com/BurntSushi/toml"
	"golang.org/x/xerrors"
)

type StorageNode struct {
}

func DefaultNode() *StorageNode {
	return &StorageNode{}
}

func NodeBytes(cfg interface{}) ([]byte, error) {
	buf := new(bytes.Buffer)
	e := toml.NewEncoder(buf)
	if err := e.Encode(cfg); err != nil {
		return nil, xerrors.Errorf("encoding node config: %w", err)
	}

	return []byte(buf.String()), nil
}
