package order

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/ipfs/go-cid"
	"github.com/mitchellh/go-homedir"
)

func StageShard(basedir string, creator string, cid cid.Cid, content []byte) error {
	// TODO: check enough space
	// TODO: check existence
	path, err := homedir.Expand(basedir)
	if err != nil {
		return err
	}

	err = os.MkdirAll(filepath.Join(path, creator), 0755)
	if err != nil && !os.IsExist(err) {
		return err
	}

	//filename := fmt.Sprintf("%d-%v", orderId, cid)
	filename := fmt.Sprintf("%v", cid)
	log.Info("path: ", path)
	log.Info("staging filename: ", filename)
	file, err := os.Create(filepath.Join(path, creator, filename))
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

func GetStagedShard(basedir string, creator string, cid cid.Cid) ([]byte, error) {
	path, err := homedir.Expand(basedir)
	if err != nil {
		return nil, err
	}

	filename := fmt.Sprintf("%v", cid)
	bytes, err := os.ReadFile(filepath.Join(path, creator, filename))
	if err != nil {
		return nil, err
	} else {
		return bytes, nil
	}
}

func UnstageShard(basedir string, creator string, cid cid.Cid) error {
	path, err := homedir.Expand(basedir)
	if err != nil {
		return err
	}

	filename := fmt.Sprintf("%v", cid)
	return os.Remove(filepath.Join(path, creator, filename))
}
