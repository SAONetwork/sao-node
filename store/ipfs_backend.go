package store

import (
	"fmt"
	shell "github.com/SaoNetwork/go-ipfs-api"
	"github.com/ipfs/go-cid"
	"golang.org/x/xerrors"
	"io"
	"strings"
)

type IpfsBackend struct {
	ipfsAddress string
	ipfsApi     *shell.Shell
}

func NewIpfsBackend(connectionString string) (*IpfsBackend, error) {
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
	b.ipfsApi = shell.NewShell(b.ipfsAddress)
	return nil
}

func (b *IpfsBackend) Close() error {
	return nil
}

func (b *IpfsBackend) Store(reader io.Reader) (any, error) {
	hash, err := b.ipfsApi.Add(reader, shell.Pin(true), shell.CidVersion(1))
	log.Debugf("%s store hash: %v", b.Id(), hash)
	return hash, err
}

func (b *IpfsBackend) Get(cid cid.Cid) (io.ReadCloser, error) {
	return b.ipfsApi.Cat(cid.String())
}

func (b *IpfsBackend) Remove(cid cid.Cid) error {
	return nil
}
