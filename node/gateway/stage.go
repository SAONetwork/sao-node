package gateway

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/SaoNetwork/sao-node/types"

	saotypes "github.com/SaoNetwork/sao/x/sao/types"
	"github.com/ipfs/go-cid"
	"github.com/mitchellh/go-homedir"
)

func StageShard(basedir string, orderProposal saotypes.Proposal, content []byte) (string, error) {
	// TODO: check enough space
	// TODO: check existence
	path, err := homedir.Expand(basedir)
	if err != nil {
		return "", types.Wrapf(types.ErrInvalidPath, "%s", basedir)
	}

	err = os.MkdirAll(filepath.Join(path, orderProposal.Owner), 0755)
	if err != nil && !os.IsExist(err) {
		return "", types.Wrap(types.ErrCreateDirFailed, err)
	}

	//filename := fmt.Sprintf("%d-%v", orderId, cid)
	filename := fmt.Sprintf("%s-%s", orderProposal.Cid, orderProposal.DataId)
	log.Debugf("staging file: %s/%s/%s", path, orderProposal.Owner, filename)
	filepath := filepath.Join(path, orderProposal.Owner, filename)
	file, err := os.Create(filepath)
	if err != nil {
		return "", types.Wrap(types.ErrCreateFileFailed, err)
	}
	defer file.Close()

	_, err = file.Write(content)
	if err != nil {
		return "", types.Wrap(types.ErrWriteFileFailed, err)
	}
	return filepath, nil
}

func GetStagedShard(basedir string, creator string, cid cid.Cid, dataId string) ([]byte, error) {
	path, err := homedir.Expand(basedir)
	if err != nil {
		return nil, types.Wrapf(types.ErrInvalidPath, "%s", basedir)
	}

	filename := cid.String() + "-" + dataId
	bytes, err := os.ReadFile(filepath.Join(path, creator, filename))
	if err != nil {
		return nil, types.Wrap(types.ErrReadFileFailed, err)
	} else {
		return bytes, nil
	}
}

func UnstageShard(basedir string, creator string, cid string, dataId string) error {
	path, err := homedir.Expand(basedir)
	if err != nil {
		return types.Wrapf(types.ErrInvalidPath, "%s", basedir)
	}

	return os.Remove(filepath.Join(path, creator, fmt.Sprintf("%s-%s", cid, dataId)))
}
