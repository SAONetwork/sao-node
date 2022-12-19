package types

import (
	"bytes"
	"encoding/json"
	"io"

	"github.com/ipfs/go-cid"
	"golang.org/x/xerrors"
)

const (
	ShardLoadProtocol  = "/sao/load/shard/1.0"
	ShardStoreProtocol = "/sao/store/shard/1.0"

	ErrorCodeInvalidRequest       = 1
	ErrorCodeInvalidTx            = 2
	ErrorCodeInternalErr          = 3
	ErrorCodeInvalidProvider      = 4
	ErrorCodeInvalidShardCid      = 5
	ErrorCodeInvalidOrderProvider = 6
	ErrorCodeInvalidShardAssignee = 7
)

type ShardStaging struct {
	Basedir string
}

// TODO: store node should sign the request.
type ShardReq struct {
	Owner    string
	OrderId  uint64
	Cid      cid.Cid
	Proposal *MetadataProposal
}

type ShardResp struct {
	OrderId uint64
	Cid     cid.Cid
	Content []byte
}

func (f *ShardReq) Unmarshal(r io.Reader, format string) (err error) {
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

func (f *ShardReq) Marshal(w io.Writer, format string) error {
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

func (f *ShardResp) Marshal(w io.Writer, format string) error {
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

func (f *ShardResp) Unmarshal(r io.Reader, format string) (err error) {
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
