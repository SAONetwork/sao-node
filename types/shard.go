package types

import (
	"bytes"
	"encoding/json"
	"io"

	"github.com/ipfs/go-cid"
	"golang.org/x/xerrors"
)

const (
	ShardStoreProtocol = "/sao/store/shard/1.0"
)

type ShardStaging struct {
	Basedir string
}

// TODO: store node should sign the request.
type ShardStoreReq struct {
	Owner   string
	OrderId uint64
	Cid     cid.Cid
}

type ShardStoreResp struct {
	OrderId uint64
	Cid     cid.Cid
	Content []byte
}

func (f *ShardStoreReq) Unmarshal(r io.Reader, format string) (err error) {
	if format == "json" {
		buf := &bytes.Buffer{}
		buf.ReadFrom(r)
		err = json.Unmarshal(buf.Bytes(), f)
		if err != nil {
			return err
		}
	} else {
		// TODO: CBOR marshal
		return xerrors.Errorf("not implemented yet")
	}
	return nil
}

func (f *ShardStoreReq) Marshal(w io.Writer, format string) error {
	if format == "json" {
		bytes, err := json.Marshal(f)
		if err != nil {
			return err
		}
		_, err = w.Write(bytes)
		if err != nil {
			return err
		}
	} else {
		// TODO: CBOR marshal
		return xerrors.Errorf("not implemented yet")
	}
	return nil
}

func (f *ShardStoreResp) Marshal(w io.Writer, format string) error {
	if format == "json" {
		bytes, err := json.Marshal(f)
		if err != nil {
			return err
		}
		_, err = w.Write(bytes)
		if err != nil {
			return err
		}
	} else {
		// TODO: CBOR marshal
		return xerrors.Errorf("not implemented yet")
	}
	return nil
}

func (f *ShardStoreResp) Unmarshal(r io.Reader, format string) (err error) {
	if format == "json" {
		buf := &bytes.Buffer{}
		buf.ReadFrom(r)
		err = json.Unmarshal(buf.Bytes(), f)
		if err != nil {
			return err
		}
	} else {
		// TODO: CBOR marshal
		return xerrors.Errorf("not implemented yet")
	}
	return nil
}
