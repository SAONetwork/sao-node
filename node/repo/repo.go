package repo

import (
	"context"
	"crypto/rand"
	"errors"
	"os"
	"path/filepath"
	"sao-node/node/config"
	"sao-node/types"
	"sao-node/utils"
	"sync"

	"github.com/ipfs/go-datastore"
	logging "github.com/ipfs/go-log/v2"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/mitchellh/go-homedir"
)

var log = logging.Logger("repo")

var ErrRepoExists = types.Wrapf(types.ErrStatFailed, "repo exists")

const (
	fsConfig    = "config.toml"
	fsKeystore  = "keystore"
	fsLibp2pKey = "libp2p.key"
	fsDatastore = "datastore"
)

var (
	ErrNoAPIEndpoint = errors.New("API not running (no endpoint)")
)

type Repo struct {
	Path       string
	configPath string

	readonly bool

	ds     map[string]datastore.Batching
	dsErr  error
	dsOnce sync.Once
}

func PrepareRepo(repoPath string) (*Repo, error) {
	repo, err := NewRepo(repoPath)
	if err != nil {
		return nil, err
	}

	ok, err := repo.Exists()
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, types.Wrapf(types.ErrInvalidRepoPath, "repo at '%s' is not initialized, run 'saonode init' to set it up", repoPath)
	}
	return repo, nil
}

func NewRepo(path string) (*Repo, error) {
	path, err := homedir.Expand(path)
	if err != nil {
		return nil, types.Wrap(types.ErrInvalidRepoPath, err)
	}

	return &Repo{
		Path:       path,
		configPath: filepath.Join(path, fsConfig),
	}, nil
}

func (r *Repo) Exists() (bool, error) {
	_, err := os.Stat(filepath.Join(r.Path, fsKeystore))
	notexist := os.IsNotExist(err)
	if notexist {
		return false, nil
	} else {
		if err != nil {
			return true, types.Wrap(types.ErrOpenFileFailed, err)
		} else {
			return true, nil
		}
	}
}

func (r *Repo) Init(chainAddress string, TxPoolSize uint) error {
	exist, err := r.Exists()
	if err != nil {
		return types.Wrap(types.ErrOpenRepoFailed, err)
	}
	if exist {
		return nil
	}

	log.Infof("Initializing repo at '%s'", r.Path)
	err = os.MkdirAll(r.Path, 0755) //nolint: gosec
	if err != nil && !os.IsExist(err) {
		return types.Wrap(types.ErrOpenFileFailed, err)
	}

	if err := r.initConfig(chainAddress, TxPoolSize); err != nil {
		return types.Wrapf(types.ErrInitRepoFailed, "init config: %v", err)
	}
	err = r.initKeystore()
	if err != nil {
		return types.Wrap(types.ErrInitRepoFailed, err)
	}

	_, err = r.GeneratePeerId()
	if err != nil {
		return types.Wrap(types.ErrInitRepoFailed, err)
	}

	return nil
}

func (r *Repo) GeneratePeerId() (crypto.PrivKey, error) {
	pk, _, err := crypto.GenerateEd25519Key(rand.Reader)
	if err != nil {
		return nil, err
	}

	kbytes, err := crypto.MarshalPrivateKey(pk)
	if err != nil {
		return nil, err
	}

	err = r.setPeerId(kbytes)
	if err != nil {
		return nil, err
	}

	return pk, nil
}

func (r *Repo) GetKeyBytes() ([]byte, error) {
	libp2pPath := filepath.Join(r.Path, fsKeystore, fsLibp2pKey)
	key, err := os.ReadFile(libp2pPath)
	if err != nil {
		return nil, types.Wrap(types.ErrReadConfigFailed, err)
	}
	return key, nil
}

func (r *Repo) PeerId() (crypto.PrivKey, error) {
	libp2pPath := filepath.Join(r.Path, fsKeystore, fsLibp2pKey)
	key, err := os.ReadFile(libp2pPath)
	if err != nil {
		return nil, types.Wrap(types.ErrReadConfigFailed, err)
	}
	return crypto.UnmarshalPrivateKey(key)
}

func (r *Repo) setPeerId(data []byte) error {
	libp2pPath := filepath.Join(r.Path, fsKeystore, fsLibp2pKey)
	err := os.WriteFile(libp2pPath, data, 0600)
	if err != nil {
		return err
	}
	return nil
}

func (r *Repo) Config() (interface{}, error) {
	return utils.FromFile(r.configPath, r.defaultConfig())
}

func (r *Repo) Datastore(ctx context.Context, ns string) (datastore.Batching, error) {
	r.dsOnce.Do(func() {
		r.ds, r.dsErr = r.openDatastores(r.readonly)
	})

	if r.dsErr != nil {
		return nil, types.Wrap(types.ErrOpenDataStoreFailed, r.dsErr)
	}
	ds, ok := r.ds[ns]
	if ok {
		return ds, nil
	}
	return nil, types.Wrapf(types.ErrOpenDataStoreFailed, "no such datastore: %s", ns)
}

func (r *Repo) initConfig(chainAddress string, TxPoolSize uint) error {
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

	defaultConfig := r.defaultConfig()
	newConfig := r.defaultConfig()
	if chainAddress != "" {
		newConfig.Chain.Remote = chainAddress
	}
	newConfig.Chain.TxPoolSize = TxPoolSize

	// comm, err := config.ConfigComment(defaultConfig)
	comm, err := config.ConfigUpdate(newConfig, defaultConfig, true)
	//comm, err := utils.NodeBytes(r.defaultConfig(chainAddress))
	if err != nil {
		return types.Wrapf(types.ErrReadConfigFailed, "load default: %v", err)
	}
	_, err = c.Write(comm)
	if err != nil {
		return types.Wrapf(types.ErrWriteConfigFailed, "write config: %v", err)
	}

	if err := c.Close(); err != nil {
		return types.Wrapf(types.ErrCloseFileFailed, "close config: %v", err)
	}
	return nil
}

func (r *Repo) defaultConfig() *config.Node {
	repo := config.DefaultSaoNode()
	return repo
}

func (r *Repo) initKeystore() error {
	kstorePath := filepath.Join(r.Path, fsKeystore)
	if _, err := os.Stat(kstorePath); err == nil {
		return ErrRepoExists
	} else if !os.IsNotExist(err) {
		return err
	}
	return os.Mkdir(kstorePath, 0700)
}

// join joins path elements with fsr.path
func (fsr *Repo) join(paths ...string) string {
	return filepath.Join(append([]string{fsr.Path}, paths...)...)
}
