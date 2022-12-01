package repo

import (
	"context"
	"crypto/rand"
	"errors"
	"os"
	"path/filepath"
	"sao-storage-node/node/config"
	"sao-storage-node/utils"
	"sync"

	"github.com/ipfs/go-datastore"
	logging "github.com/ipfs/go-log/v2"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/mitchellh/go-homedir"
	"golang.org/x/xerrors"
)

var log = logging.Logger("repo")

var ErrRepoExists = xerrors.New("repo exists")

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
	path       string
	configPath string

	readonly bool

	ds     map[string]datastore.Batching
	dsErr  error
	dsOnce sync.Once
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

func (r *Repo) Init(chainAddress string) error {
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

	if err := r.initConfig(chainAddress); err != nil {
		return xerrors.Errorf("init config: %w", err)
	}
	err = r.initKeystore()
	if err != nil {
		return err
	}

	_, err = r.GeneratePeerId()
	if err != nil {
		return err
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
	libp2pPath := filepath.Join(r.path, fsKeystore, fsLibp2pKey)
	key, err := os.ReadFile(libp2pPath)
	if err != nil {
		return nil, err
	}
	return key, nil
}

func (r *Repo) PeerId() (crypto.PrivKey, error) {
	libp2pPath := filepath.Join(r.path, fsKeystore, fsLibp2pKey)
	key, err := os.ReadFile(libp2pPath)
	if err != nil {
		return nil, err
	}
	return crypto.UnmarshalPrivateKey(key)
}

func (r *Repo) setPeerId(data []byte) error {
	libp2pPath := filepath.Join(r.path, fsKeystore, fsLibp2pKey)
	err := os.WriteFile(libp2pPath, data, 0600)
	if err != nil {
		return err
	}
	return nil
}

func (r *Repo) Config() (interface{}, error) {
	return utils.FromFile(r.configPath, r.defaultConfig(""))
}

func (r *Repo) Datastore(ctx context.Context, ns string) (datastore.Batching, error) {
	r.dsOnce.Do(func() {
		r.ds, r.dsErr = r.openDatastores(r.readonly)
	})

	if r.dsErr != nil {
		return nil, r.dsErr
	}
	ds, ok := r.ds[ns]
	if ok {
		return ds, nil
	}
	return nil, xerrors.Errorf("no such datastore: %s", ns)
}

func (r *Repo) initConfig(chainAddress string) error {
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

	comm, err := utils.NodeBytes(r.defaultConfig(chainAddress))
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

func (r *Repo) defaultConfig(chainAddress string) interface{} {
	repo := config.DefaultSaoNode()
	repo.Chain.Remote = chainAddress
	return repo
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

// join joins path elements with fsr.path
func (fsr *Repo) join(paths ...string) string {
	return filepath.Join(append([]string{fsr.path}, paths...)...)
}
