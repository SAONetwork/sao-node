package types

import (
	"bytes"
	"encoding/json"
	"io"

	"github.com/ipfs/go-cid"
)

type AssignTxType string

const (
	ShardAssignProtocol   = "/sao/shard/assign/1.0"
	ShardCompleteProtocol = "/sao/shard/complete/1.0"

	AssignTxTypeStore AssignTxType = "MsgStore"
	AssignTxTypeReady AssignTxType = "MsgReady"
)

type ShardAssignReq struct {
	OrderId      uint64
	Assignee     string
	TxHash       string
	Height       int64
	AssignTxType AssignTxType
}

func (f *ShardAssignReq) Unmarshal(r io.Reader, format string) (err error) {
	if format == "json" {
		buf := &bytes.Buffer{}
		buf.ReadFrom(r)
		err = json.Unmarshal(buf.Bytes(), f)
		if err != nil {
			return err
		}
	} else {
		// TODO: CBOR marshal
		return Wrap(ErrUnSupport, nil)
	}
	return nil
}

func (f *ShardAssignReq) Marshal(w io.Writer, format string) error {
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
		return Wrap(ErrUnSupport, nil)
	}
	return nil
}

type ShardAssignResp struct {
	Code    uint64
	Message string
}

func (f *ShardAssignResp) Unmarshal(r io.Reader, format string) (err error) {
	if format == "json" {
		buf := &bytes.Buffer{}
		buf.ReadFrom(r)
		err = json.Unmarshal(buf.Bytes(), f)
		if err != nil {
			return err
		}
	} else {
		// TODO: CBOR marshal
		return Wrap(ErrUnSupport, nil)
	}
	return nil
}

func (f *ShardAssignResp) Marshal(w io.Writer, format string) error {
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
		return Wrap(ErrUnSupport, nil)
	}
	return nil
}

type ShardCompleteReq struct {
	OrderId uint64
	Cids    []cid.Cid
	TxHash  string
	Height  int64
	Code    uint64
	Message string
}

func (f *ShardCompleteReq) Unmarshal(r io.Reader, format string) (err error) {
	if format == "json" {
		buf := &bytes.Buffer{}
		buf.ReadFrom(r)
		err = json.Unmarshal(buf.Bytes(), f)
		if err != nil {
			return err
		}
	} else {
		// TODO: CBOR marshal
		return Wrap(ErrUnSupport, nil)
	}
	return nil
}

func (f *ShardCompleteReq) Marshal(w io.Writer, format string) error {
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
		return Wrap(ErrUnSupport, nil)
	}
	return nil
}

type ShardCompleteResp struct {
	Code    uint64
	Message string
}

func (f *ShardCompleteResp) Unmarshal(r io.Reader, format string) (err error) {
	if format == "json" {
		buf := &bytes.Buffer{}
		buf.ReadFrom(r)
		err = json.Unmarshal(buf.Bytes(), f)
		if err != nil {
			return err
		}
	} else {
		// TODO: CBOR marshal
		return Wrap(ErrUnSupport, nil)
	}
	return nil
}

func (f *ShardCompleteResp) Marshal(w io.Writer, format string) error {
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
		return Wrap(ErrUnSupport, nil)
	}
	return nil
}
