package node

import (
	"fmt"
	"github.com/ipfs/go-cid"
	"github.com/mitchellh/go-homedir"
	"os"
	"path/filepath"
)

type ShardStaging struct {
	basedir string
}

func NewShardStaging(basedir string) ShardStaging {
	return ShardStaging{
		basedir: basedir,
	}
}

func (ss *ShardStaging) StageShard(orderId uint64, cid cid.Cid, content []byte) error {
	// TODO: check enough space
	// TODO: check existence
	path, err := homedir.Expand(ss.basedir)
	if err != nil {
		return err
	}

	filename := fmt.Sprintf("%d-%v", orderId, cid)
	file, err := os.Create(filepath.Join(path, filename))
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.Write(content)
	if err != nil {
		return err
	}
	return nil
}

func (ss *ShardStaging) GetStagedShard(orderId uint64, cid cid.Cid) ([]byte, error) {
	path, err := homedir.Expand(ss.basedir)
	if err != nil {
		return nil, err
	}

	filename := fmt.Sprintf("%d-%v", orderId, cid)
	return os.ReadFile(filepath.Join(path, filename))
}
