package store

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/ipfs/go-cid"
	files "github.com/ipfs/go-ipfs-files"
	httpapi "github.com/ipfs/go-ipfs-http-client"
	icore "github.com/ipfs/interface-go-ipfs-core"
	"github.com/ipfs/interface-go-ipfs-core/options"
	icorepath "github.com/ipfs/interface-go-ipfs-core/path"
	ma "github.com/multiformats/go-multiaddr"
	"golang.org/x/xerrors"
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
		return nil, xerrors.Errorf("unsupported ipfs connection protocol")
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
		return err
	}
	api, err := httpapi.NewApi(addr)
	if err != nil {
		return err
	}

	b.api = api
	return err
}

func (b *IpfsBackend) Close() error {
	return nil
}

func (b *IpfsBackend) Store(ctx context.Context, reader io.Reader) (any, error) {
	r, err := b.api.Unixfs().Add(ctx, files.NewReaderFile(reader), options.Unixfs.Pin(true), options.Unixfs.CidVersion(1))
	if err != nil {
		return nil, err
	}

	//hash, err := b.ipfsApi.Add(reader, shell.Pin(true), shell.CidVersion(1))
	log.Debugf("%s store hash: %v", b.Id(), r.String())
	return r.String(), err
}

func (b *IpfsBackend) IsExist(ctx context.Context, cid cid.Cid) (bool, error) {
	path := icorepath.New(cid.String())
	r, err := b.api.Unixfs().Get(ctx, path)
	if err != nil {
		return false, err
	}
	return r != nil, nil
}

func (b *IpfsBackend) Get(ctx context.Context, cid cid.Cid) (io.ReadCloser, error) {
	path := icorepath.New(cid.String())
	r, err := b.api.Unixfs().Get(ctx, path)
	if err != nil {
		return nil, err
	}
	//return b.ipfsApi.Cat(cid.String())
	return r.(files.File), nil
}

func (b *IpfsBackend) Remove(ctx context.Context, cid cid.Cid) error {
	path := icorepath.New(cid.String())
	return b.api.Pin().Rm(ctx, path)
}
