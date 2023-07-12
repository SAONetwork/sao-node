package store

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/SaoNetwork/sao-node/types"

	"github.com/ipfs/go-cid"
	files "github.com/ipfs/go-ipfs-files"
	httpapi "github.com/ipfs/go-ipfs-http-client"
	icore "github.com/ipfs/interface-go-ipfs-core"
	"github.com/ipfs/interface-go-ipfs-core/options"
	icorepath "github.com/ipfs/interface-go-ipfs-core/path"
	ma "github.com/multiformats/go-multiaddr"
)

type IpfsBackend struct {
	ipfsAddress string
	//ipfsApi     *shell.Shell
	api icore.CoreAPI
}

func NewIpfsBackend(connectionString string, api icore.CoreAPI) (*IpfsBackend, error) {
	if strings.HasPrefix(connectionString, "ipfs+sao") {
		return &IpfsBackend{
			ipfsAddress: connectionString,
			api:         api,
		}, nil
	}

	var conn string
	if strings.HasPrefix(connectionString, "ipfs+ma:") {
		conn = strings.Replace(connectionString, "ipfs+ma:", "", 1)
		// }
		// else if strings.HasPrefix(connectionString, "ipfs+https") {
		// 	conn = strings.Replace(connectionString, "ipfs+https", "https", 1)
	} else {
		return nil, types.Wrap(types.ErrUnSupportProtocol, nil)
	}

	b := IpfsBackend{
		ipfsAddress: conn,
	}
	return &b, nil
}

func (b *IpfsBackend) Id() string {
	return fmt.Sprintf("%s-%s", b.Type(), b.ipfsAddress)
}

func (b *IpfsBackend) Type() string {
	return "ipfs"
}

func (b *IpfsBackend) Open() error {
	//b.ipfsApi = shell.NewShell(b.ipfsAddress)
	if strings.HasPrefix(b.ipfsAddress, "ipfs+sao") {
		return nil
	}

	addr, err := ma.NewMultiaddr(b.ipfsAddress)
	if err != nil {
		return types.Wrap(types.ErrOpenIpfsBackendFailed, err)
	}
	api, err := httpapi.NewApi(addr)
	if err != nil {
		return types.Wrap(types.ErrCreateIpfsApiServiceFailed, err)
	}

	b.api = api
	return nil
}

func (b *IpfsBackend) Close() error {
	return nil
}

func (b *IpfsBackend) Store(ctx context.Context, reader io.Reader) (any, error) {
	// r, err := b.api.Unixfs().Add(ctx, files.NewReaderFile(reader), options.Unixfs.Pin(true), options.Unixfs.CidVersion(1))
	blkSt, err := b.api.Block().Put(ctx, files.NewReaderFile(reader), options.Block.Pin(true), func(settings *options.BlockPutSettings) error {
		settings.CidPrefix.Version = 1
		return nil
	})
	if err != nil {
		return nil, types.Wrap(types.ErrStoreFailed, err)
	}

	//hash, err := b.ipfsApi.Add(reader, shell.Pin(true), shell.CidVersion(1))
	// log.Debugf("%s store hash: %s %v", b.Id(), r.String(), r.Cid())
	// log.Debugf("codec:%v", r.Cid().Type())
	log.Debugf("%s store hash: %v %v", b.Id(), blkSt.Path().Cid().Version(), blkSt.Path().Cid().Type())
	fmt.Printf("%s store hash: %v %v", b.Id(), blkSt.Path().Cid().Version(), blkSt.Path().Cid().Type())
	return blkSt.Path().Cid().String(), nil
}

func (b *IpfsBackend) IsExist(ctx context.Context, cid cid.Cid) (bool, error) {
	path := icorepath.New(cid.String())
	s, err := b.api.Block().Stat(ctx, path)
	// r, err := b.api.Unixfs().Get(ctx, path)
	if err != nil {
		return false, types.Wrap(types.ErrStatFailed, err)
	}
	err = s.Path().IsValid()
	if err != nil {
		return false, types.Wrap(types.ErrInvalidPath, err)
	}
	// return r != nil, nil
	return true, nil
}

func (b *IpfsBackend) Get(ctx context.Context, cid cid.Cid) (io.Reader, error) {
	path := icorepath.New(cid.String())
	// r, err := b.api.Unixfs().Get(ctx, path)
	r, err := b.api.Block().Get(ctx, path)
	if err != nil {
		return nil, types.Wrap(types.ErrGetFailed, err)
	}
	//return b.ipfsApi.Cat(cid.String())
	return r, nil
}

func (b *IpfsBackend) Remove(ctx context.Context, cid cid.Cid) error {
	path := icorepath.New(cid.String())
	return b.api.Pin().Rm(ctx, path)
}
