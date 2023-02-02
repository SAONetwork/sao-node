package store

import (
	"context"
	"io"
	"sao-node/types"

	"github.com/ipfs/go-cid"
	logging "github.com/ipfs/go-log/v2"
)

var log = logging.Logger("store")

type StoreBackend interface {
	Id() string
	Type() string
	Open() error
	Close() error
	Store(ctx context.Context, reader io.Reader) (any, error)
	Remove(ctx context.Context, cid cid.Cid) error
	Get(ctx context.Context, cid cid.Cid) (io.Reader, error)
	IsExist(ctx context.Context, cid cid.Cid) (bool, error)
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

func (ss *StoreManager) Store(ctx context.Context, cid cid.Cid, reader io.Reader) (any, error) {
	var err error
	for _, back := range ss.backends {
		_, err = back.Store(ctx, reader)
		if err != nil {
			log.Errorf("%s store error: %v", back.Id(), err)
		} else {
			err = nil
		}
	}
	return nil, err
}

func (ss *StoreManager) Remove(ctx context.Context, cid cid.Cid) error {
	var err error
	for _, back := range ss.backends {
		err = back.Remove(ctx, cid)
		if err != nil {
			log.Errorf("%s remove cid=%v error: %v", back.Id(), cid, err)
		} else {
			err = nil
		}
	}
	return err
}

func (ss *StoreManager) Get(ctx context.Context, cid cid.Cid) (io.Reader, error) {
	for _, back := range ss.backends {
		reader, err := back.Get(ctx, cid)
		if err != nil {
			log.Errorf("%s get cid=%v error: %v", back.Id(), cid, err)
			continue
		}
		return reader, nil
	}
	return nil, types.Wrapf(types.ErrGetFailed, "failed to get cid %s", cid)
}

func (ss *StoreManager) IsExist(ctx context.Context, cid cid.Cid) bool {
	for _, back := range ss.backends {
		isExist, err := back.IsExist(ctx, cid)
		if err != nil {
			log.Errorf("%s get cid=%v error: %v", back.Id(), cid, err)
			continue
		}

		if !isExist {
			log.Errorf("%s get cid=%v error: not found", back.Id(), cid)
			continue
		}
		return true
	}

	return false
}
