package gateway

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/ipfs/go-cid"
	"github.com/mitchellh/go-homedir"
)

func StageShard(basedir string, creator string, cid string, content []byte) (string, error) {
	// TODO: check enough space
	// TODO: check existence
	path, err := homedir.Expand(basedir)
	if err != nil {
		return "", err
	}

	err = os.MkdirAll(filepath.Join(path, creator), 0755)
	if err != nil && !os.IsExist(err) {
		return "", err
	}

	//filename := fmt.Sprintf("%d-%v", orderId, cid)
	filename := fmt.Sprintf("%v", cid)
	log.Debugf("staging file: %s/%s/%s", path, creator, filename)
	filepath := filepath.Join(path, creator, filename)
	file, err := os.Create(filepath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	_, err = file.Write(content)
	if err != nil {
		return "", err
	}
	return filepath, nil
}

func GetStagedShard(basedir string, creator string, cid cid.Cid) ([]byte, error) {
	path, err := homedir.Expand(basedir)
	if err != nil {
		return nil, err
	}

	filename := cid.String()
	bytes, err := os.ReadFile(filepath.Join(path, creator, filename))
	if err != nil {
		return nil, err
	} else {
		return bytes, nil
	}
}

func UnstageShard(basedir string, creator string, cid string) error {
	path, err := homedir.Expand(basedir)
	if err != nil {
		return err
	}

	return os.Remove(filepath.Join(path, creator, cid))
}
