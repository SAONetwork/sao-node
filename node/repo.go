package node

import (
	logging "github.com/ipfs/go-log/v2"
	"github.com/mitchellh/go-homedir"
	"golang.org/x/xerrors"
	"io/ioutil"
	"os"
	"path/filepath"
	"sao-storage-node/node/config"
)

var log = logging.Logger("repo")

var ErrRepoExists = xerrors.New("repo exists")

const (
	fsConfig    = "config.toml"
	fsKeystore  = "keystore"
	fsLibp2pKey = "libp2p.key"
)

type Repo struct {
	path       string
	configPath string
}

func NewRepo(path string) (*Repo, error) {
	path, err := homedir.Expand(path)
	if err != nil {
		return nil, err
	}

	return &Repo{
		path:       path,
		configPath: filepath.Join(path, fsConfig),
	}, nil
}

func (r *Repo) Exists() (bool, error) {
	// TODO:
	_, err := os.Stat(filepath.Join(r.path, fsKeystore))
	notexist := os.IsNotExist(err)
	if notexist {
		err = nil
	}
	return !notexist, err
}

func (r *Repo) Init() error {
	exist, err := r.Exists()
	if err != nil {
		return err
	}
	if exist {
		return nil
	}

	log.Infof("Initializing repo at '%s'", r.path)
	err = os.MkdirAll(r.path, 0755) //nolint: gosec
	if err != nil && !os.IsExist(err) {
		return err
	}

	if err := r.initConfig(); err != nil {
		return xerrors.Errorf("init config: %w", err)
	}
	return r.initKeystore()
}

func (r *Repo) SetPeerId(data []byte) error {
	libp2pPath := filepath.Join(r.path, fsKeystore, fsLibp2pKey)
	err := ioutil.WriteFile(libp2pPath, data, 0600)
	if err != nil {
		return err
	}
	return nil
}

func (r *Repo) initConfig() error {
	_, err := os.Stat(r.configPath)
	if err == nil {
		// exists
		return nil
	} else if !os.IsNotExist(err) {
		return err
	}

	c, err := os.Create(r.configPath)
	if err != nil {
		return err
	}

	comm, err := config.NodeBytes(r.defaultConfig())
	if err != nil {
		return xerrors.Errorf("load default: %w", err)
	}
	_, err = c.Write(comm)
	if err != nil {
		return xerrors.Errorf("write config: %w", err)
	}

	if err := c.Close(); err != nil {
		return xerrors.Errorf("close config: %w", err)
	}
	return nil
}

func (r *Repo) defaultConfig() interface{} {
	return config.DefaultNode()
}

func (r *Repo) initKeystore() error {
	kstorePath := filepath.Join(r.path, fsKeystore)
	if _, err := os.Stat(kstorePath); err == nil {
		return ErrRepoExists
	} else if !os.IsNotExist(err) {
		return err
	}
	return os.Mkdir(kstorePath, 0700)
}
