package types

import (
	"bytes"
	"encoding/json"
	"io"

	"github.com/ipfs/go-cid"
)

type AssignTxType string

const (
	ShardLoadProtocol     = "/sao/shard/load/1.0"
	ShardStoreProtocol    = "/sao/shard/store/1.0"
	ShardAssignProtocol   = "/sao/shard/assign/1.0"
	ShardCompleteProtocol = "/sao/shard/complete/1.0"
	ShardMigrateProtocol  = "/sao/shard/migrate/1.0"
	ShardPingPongProtocol = "/sao/shard/pingpong/1.0"
	RpcProtocol           = "/sao/rpc/1.0"
	PublicIpProtocol      = "/sao/ip/1.0"

	ErrorCodeInvalidRequest       = 1
	ErrorCodeInvalidTx            = 2
	ErrorCodeInternalErr          = 3
	ErrorCodeInvalidProvider      = 4
	ErrorCodeInvalidShardCid      = 5
	ErrorCodeInvalidOrderProvider = 6
	ErrorCodeInvalidShardAssignee = 7

	AssignTxTypeStore AssignTxType = "MsgStore"
	AssignTxTypeReady AssignTxType = "MsgReady"

	FormatJson string = "json"
	FormatCbor string = "cbor"
)

type ShardStaging struct {
	Basedir string
}

// TODO: store node should sign the request.
type ShardLoadReq struct {
	Owner         string
	DataId        string
	OrderId       uint64
	Cid           cid.Cid
	Proposal      MetadataProposalCbor
	RequestId     int64
	RelayProposal RelayProposalCbor
}

type ShardLoadResp struct {
	Code       uint64
	Message    string
	OrderId    uint64
	Cid        cid.Cid
	Content    []byte
	RequestId  int64
	ResponseId int64
}

type ShardAssignReq struct {
	OrderId       uint64
	DataId        string
	Assignee      string
	TxHash        string
	Height        int64
	AssignTxType  AssignTxType
	TimeoutHeight uint64
}

type ShardAssignResp struct {
	Code    uint64
	Message string
}

type ShardCompleteReq struct {
	OrderId uint64
	DataId  string
	Cids    []cid.Cid
	TxHash  string
	Height  int64
}

type ShardCompleteResp struct {
	Code        uint64
	Message     string
	Recoverable bool // if can handle this shard after retry
}

type ShardMigrateReq struct {
	MigrateFrom string
	OrderId     uint64
	DataId      string
	TxHash      string
	TxHeight    int64
	Cid         string
	Content     []byte
}

type ShardMigrateResp struct {
	Code           uint64
	Message        string
	CompleteHash   string
	CompleteHeight int64
}

type ShardPingPong struct {
	Local string
}

func (f *ShardMigrateReq) Unmarshal(r io.Reader, format string) error {
	var err error
	if format == FormatJson {
		buf := &bytes.Buffer{}
		buf.ReadFrom(r)
		err = json.Unmarshal(buf.Bytes(), f)
	} else {
		err = f.UnmarshalCBOR(r)
	}
	return err
}

func (f *ShardMigrateReq) Marshal(w io.Writer, format string) error {
	var err error
	if format == FormatJson {
		bytes, err := json.Marshal(f)
		if err != nil {
			return err
		}
		_, err = w.Write(bytes)
	} else {
		err = f.MarshalCBOR(w)
	}
	return err
}

func (f *ShardMigrateResp) Unmarshal(r io.Reader, format string) error {
	var err error
	if format == FormatJson {
		buf := &bytes.Buffer{}
		buf.ReadFrom(r)
		err = json.Unmarshal(buf.Bytes(), f)
	} else {
		err = f.UnmarshalCBOR(r)
	}
	return err
}

func (f *ShardMigrateResp) Marshal(w io.Writer, format string) error {
	var err error
	if format == FormatJson {
		bytes, err := json.Marshal(f)
		if err != nil {
			return err
		}
		_, err = w.Write(bytes)
	} else {
		err = f.MarshalCBOR(w)
	}
	return err
}

func (f *ShardLoadReq) Marshal(w io.Writer, format string) error {
	var err error
	if format == FormatJson {
		bytes, err := json.Marshal(f)
		if err != nil {
			return err
		}
		_, err = w.Write(bytes)
	} else {
		err = f.MarshalCBOR(w)
	}
	return err
}
func (f *ShardLoadReq) Unmarshal(r io.Reader, format string) error {
	var err error
	if format == FormatJson {
		buf := &bytes.Buffer{}
		buf.ReadFrom(r)
		err = json.Unmarshal(buf.Bytes(), f)
		if err != nil {
			return err
		}
	} else {
		err = f.UnmarshalCBOR(r)
	}
	return err
}
func (f *ShardLoadResp) Marshal(w io.Writer, format string) error {
	var err error
	if format == FormatJson {
		bytes, err := json.Marshal(f)
		if err != nil {
			return err
		}
		_, err = w.Write(bytes)
	} else {
		err = f.MarshalCBOR(w)
	}
	return err
}

func (f *ShardLoadResp) Unmarshal(r io.Reader, format string) error {
	var err error
	if format == FormatJson {
		buf := &bytes.Buffer{}
		buf.ReadFrom(r)
		err = json.Unmarshal(buf.Bytes(), f)
		if err != nil {
			return err
		}
	} else {
		err = f.UnmarshalCBOR(r)
	}
	return err
}
func (f *ShardAssignReq) Unmarshal(r io.Reader, format string) error {
	var err error
	if format == FormatJson {
		buf := &bytes.Buffer{}
		buf.ReadFrom(r)
		err = json.Unmarshal(buf.Bytes(), f)
	} else {
		err = f.UnmarshalCBOR(r)
	}
	return err
}

func (f *ShardAssignReq) Marshal(w io.Writer, format string) error {
	var err error
	if format == FormatJson {
		bytes, err := json.Marshal(f)
		if err != nil {
			return err
		}
		_, err = w.Write(bytes)
	} else {
		err = f.MarshalCBOR(w)
	}
	return err
}

func (f *ShardAssignResp) Unmarshal(r io.Reader, format string) error {
	var err error
	if format == FormatJson {
		buf := &bytes.Buffer{}
		buf.ReadFrom(r)
		err = json.Unmarshal(buf.Bytes(), f)
		if err != nil {
			return err
		}
	} else {
		err = f.UnmarshalCBOR(r)
	}
	return err
}

func (f *ShardAssignResp) Marshal(w io.Writer, format string) error {
	var err error
	if format == FormatJson {
		bytes, err := json.Marshal(f)
		if err != nil {
			return err
		}
		_, err = w.Write(bytes)
	} else {
		err = f.MarshalCBOR(w)
	}
	return err
}
func (f *ShardCompleteReq) Unmarshal(r io.Reader, format string) error {
	var err error
	if format == FormatJson {
		buf := &bytes.Buffer{}
		buf.ReadFrom(r)
		err = json.Unmarshal(buf.Bytes(), f)
		if err != nil {
			return err
		}
	} else {
		err = f.UnmarshalCBOR(r)
	}
	return err
}

func (f *ShardCompleteReq) Marshal(w io.Writer, format string) error {
	var err error
	if format == FormatJson {
		bytes, err := json.Marshal(f)
		if err != nil {
			return err
		}
		_, err = w.Write(bytes)
		if err != nil {
			return err
		}
	} else {
		err = f.MarshalCBOR(w)
	}
	return err
}

func (f *ShardCompleteResp) Unmarshal(r io.Reader, format string) error {
	var err error
	if format == FormatJson {
		buf := &bytes.Buffer{}
		buf.ReadFrom(r)
		err = json.Unmarshal(buf.Bytes(), f)
		if err != nil {
			return err
		}
	} else {
		err = f.UnmarshalCBOR(r)
	}
	return err
}

func (f *ShardCompleteResp) Marshal(w io.Writer, format string) error {
	var err error
	if format == FormatJson {
		bytes, err := json.Marshal(f)
		if err != nil {
			return err
		}
		_, err = w.Write(bytes)
		if err != nil {
			return err
		}
	} else {
		err = f.MarshalCBOR(w)
	}
	return err
}

func (f *ShardPingPong) Unmarshal(r io.Reader, format string) error {
	var err error
	if format == FormatJson {
		buf := &bytes.Buffer{}
		buf.ReadFrom(r)
		err = json.Unmarshal(buf.Bytes(), f)
	} else {
		err = f.UnmarshalCBOR(r)
	}
	return err
}

func (f *ShardPingPong) Marshal(w io.Writer, format string) error {
	var err error
	if format == FormatJson {
		bytes, err := json.Marshal(f)
		if err != nil {
			return err
		}
		_, err = w.Write(bytes)
	} else {
		err = f.MarshalCBOR(w)
	}
	return err
}
