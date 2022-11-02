package store

import (
	"github.com/ipfs/go-cid"
	logging "github.com/ipfs/go-log/v2"
	"golang.org/x/xerrors"
	"io"
)

var log = logging.Logger("store")

type StoreBackend interface {
	Id() string
	Type() string
	Open() error
	Close() error
	Store(reader io.Reader) (any, error)
	Remove(cid cid.Cid) error
	Get(cid cid.Cid) (io.ReadCloser, error)
}

type StoreManager struct {
	backends []StoreBackend
}

func NewStoreManager(initial []StoreBackend) *StoreManager {
	return &StoreManager{
		backends: initial,
	}
}

func (ss *StoreManager) AddBackend(backend StoreBackend) {
	ss.backends = append(ss.backends, backend)
}

func (ss *StoreManager) Type() string {
	return "manager"
}

func (ss *StoreManager) Open() error {
	// TODO: any backend open error will return error.
	// in error case, handle already opened backend.
	var err error
	for _, back := range ss.backends {
		err = back.Open()
		if err != nil {
			log.Errorf("%s open error: %v", back.Id(), err)
			return err
		}
	}
	return nil
}

func (ss *StoreManager) Close() error {
	var err error
	for _, back := range ss.backends {
		err = back.Close()
		if err != nil {
			log.Errorf("%s close err: %v", back.Id(), err)
			return err
		}
	}
	return nil
}

func (ss *StoreManager) Store(cid cid.Cid, reader io.Reader) (any, error) {
	var err error
	for _, back := range ss.backends {
		_, err = back.Store(reader)
		if err != nil {
			log.Errorf("%s store error: %v", back.Id(), err)
		}
	}
	return nil, nil
}

func (ss *StoreManager) Remove(cid cid.Cid) error {
	var err error
	for _, back := range ss.backends {
		err = back.Remove(cid)
		if err != nil {
			log.Errorf("%s remove cid=%v error: %v", back.Id(), cid, err)
		}
	}
	return nil
}

func (ss *StoreManager) Get(cid cid.Cid) (io.ReadCloser, error) {
	for _, back := range ss.backends {
		reader, err := back.Get(cid)
		if err != nil {
			log.Errorf("%s remove cid=%v error: %v", back.Id(), cid, err)
			continue
		}
		return reader, err
	}
	return nil, xerrors.Errorf("failed to get cid %v", cid)
}
