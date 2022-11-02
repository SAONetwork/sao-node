package store

import (
	"context"
	"fmt"
	"github.com/ipfs/go-cid"
	files "github.com/ipfs/go-ipfs-files"
	httpapi "github.com/ipfs/go-ipfs-http-client"
	icore "github.com/ipfs/interface-go-ipfs-core"
	"github.com/ipfs/interface-go-ipfs-core/options"
	icorepath "github.com/ipfs/interface-go-ipfs-core/path"
	"golang.org/x/xerrors"
	"io"
	"strings"
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
	if strings.HasPrefix(connectionString, "ipfs+http") {
		conn = strings.Replace(connectionString, "ipfs+http", "http", 1)
	} else if strings.HasPrefix(connectionString, "ipfs+https") {
		conn = strings.Replace(connectionString, "ipfs+https", "https", 1)
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

	api, err := httpapi.NewPathApi(b.ipfsAddress)
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
