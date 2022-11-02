package store

import (
	"context"
	"fmt"
	icore "github.com/ipfs/interface-go-ipfs-core"
	"github.com/ipfs/kubo/config"
	"github.com/ipfs/kubo/core"
	"github.com/ipfs/kubo/core/coreapi"
	"github.com/ipfs/kubo/core/node/libp2p"
	"github.com/ipfs/kubo/plugin/loader"
	"github.com/ipfs/kubo/repo/fsrepo"
	"github.com/mitchellh/go-homedir"
	"io"
	"os"
	"path/filepath"
	"sync"
)

type IpfsDaemon struct {
	repoPath string
}

func NewIpfsDaemon(repoPath string) IpfsDaemon {
	return IpfsDaemon{
		repoPath: repoPath,
	}
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

	node, err := createNode(ctx, d.repoPath)
	if err != nil {
		return nil, nil, err
	}

	api, err := coreapi.NewCoreAPI(node)

	return api, node, err
}

var loadPluginsOnce sync.Once

func prepareRepo(repoPath string) error {
	repoPath, err := homedir.Expand(repoPath)
	if err != nil {
		return err
	}

	stat, err := os.Stat(repoPath)
	if os.IsNotExist(err) {
		err = os.MkdirAll(repoPath, 0700)
		if err != nil {
			return err
		}
		cfg, err := config.Init(io.Discard, 2048)
		if err != nil {
			return err
		}
		err = fsrepo.Init(repoPath, cfg)
		if err != nil {
			return fmt.Errorf("failed to init ipfs repo: %s", err)
		}
	} else if !stat.IsDir() {
		return fmt.Errorf("repo %s already exists but not a dir", repoPath)
	}
	return nil
}

func setupPlugins(externalPluginsPath string) error {
	plugins, err := loader.NewPluginLoader(filepath.Join(externalPluginsPath, "plugins"))
	if err != nil {
		return fmt.Errorf("error loading plugins: %s", err)
	}

	if err := plugins.Initialize(); err != nil {
		return fmt.Errorf("error initializing plugins: %s", err)
	}
	if err := plugins.Inject(); err != nil {
		return fmt.Errorf("error initializing plugins: %s", err)
	}
	return nil
}

func createNode(ctx context.Context, repoPath string) (*core.IpfsNode, error) {
	repo, err := fsrepo.Open(repoPath)
	if err != nil {
		return nil, err
	}

	nodeOptions := &core.BuildCfg{
		Online:  true,
		Routing: libp2p.DHTOption,
		Repo:    repo,
	}
	return core.NewNode(ctx, nodeOptions)
}
