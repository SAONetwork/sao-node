package store

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"sao-node/types"
	"sync"

	icore "github.com/ipfs/interface-go-ipfs-core"
	"github.com/ipfs/kubo/config"
	"github.com/ipfs/kubo/core"
	"github.com/ipfs/kubo/core/coreapi"
	"github.com/ipfs/kubo/core/node/libp2p"
	"github.com/ipfs/kubo/plugin/loader"
	"github.com/ipfs/kubo/repo/fsrepo"
	"github.com/mitchellh/go-homedir"
)

type IpfsDaemon struct {
	repoPath string
}

func NewIpfsDaemon(repoPath string) (*IpfsDaemon, error) {
	repoPath, err := homedir.Expand(repoPath)
	if err != nil {
		return nil, types.Wrapf(types.ErrInvalidRepoPath, "%v", repoPath)
	}

	return &IpfsDaemon{
		repoPath: repoPath,
	}, nil
}

func (d IpfsDaemon) Start(ctx context.Context) (icore.CoreAPI, *core.IpfsNode, error) {
	var onceErr error
	loadPluginsOnce.Do(func() {
		onceErr = setupPlugins("")
	})
	if onceErr != nil {
		return nil, nil, onceErr
	}

	err := prepareRepo(d.repoPath)
	if err != nil {
		return nil, nil, err
	}

	log.Debugf("repo path: %s", d.repoPath)
	node, err := createNode(ctx, d.repoPath)
	if err != nil {
		return nil, nil, err
	}

	api, err := coreapi.NewCoreAPI(node)
	if err != nil {
		return api, node, types.Wrap(types.ErrCreateIpfsApiServiceFailed, err)
	} else {
		return api, node, nil
	}
}

var loadPluginsOnce sync.Once

func prepareRepo(repoPath string) error {
	repoPath, err := homedir.Expand(repoPath)
	if err != nil {
		return types.Wrapf(types.ErrInvalidRepoPath, ", path=%s, %v", err)
	}

	_, err = os.Stat(filepath.Join(repoPath, "config"))
	if os.IsNotExist(err) {
		err = os.MkdirAll(repoPath, 0700)
		if err != nil {
			return types.Wrap(types.ErrCreateDirFailed, err)
		}
		cfg, err := config.Init(io.Discard, 2048)
		if err != nil {
			return types.Wrap(types.ErrInitIpfsDaemonFailed, err)
		}
		err = fsrepo.Init(repoPath, cfg)
		if err != nil {
			return types.Wrap(types.ErrInitIpfsRepoFailed, err)
		}
	}

	return nil
}

func setupPlugins(externalPluginsPath string) error {
	plugins, err := loader.NewPluginLoader(filepath.Join(externalPluginsPath, "plugins"))
	if err != nil {
		return types.Wrap(types.ErrLoadPluginsFailed, err)
	}

	if err := plugins.Initialize(); err != nil {
		return types.Wrap(types.ErrInitPluginsFailed, err)
	}
	if err := plugins.Inject(); err != nil {
		return types.Wrap(types.ErrInjectPluginsFailed, err)
	}
	return nil
}

func createNode(ctx context.Context, repoPath string) (*core.IpfsNode, error) {
	repo, err := fsrepo.Open(repoPath)
	if err != nil {
		return nil, types.Wrap(types.ErrOpenRepoFailed, err)
	}

	nodeOptions := &core.BuildCfg{
		Online:  true,
		Routing: libp2p.DHTOption,
		Repo:    repo,
	}
	return core.NewNode(ctx, nodeOptions)
}
