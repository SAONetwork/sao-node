package types

import (
	"encoding/json"

	did "github.com/SaoNetwork/sao-did"
	"github.com/ipfs/go-cid"
)

type CommitHeader struct {
	Controllers []string
	Labels      []string
	Schema      string
	DataType    string // datamodel, file, record
}

type DataCommit struct {
	Content any
}

type GenesisCommit struct {
	Header  CommitHeader
	Content any
}

type RawCommit struct {
	Content any
	Header  CommitHeader
	Id      cid.Cid
	Prev    cid.Cid
}

type OrderMeta struct {
	Owner                 string
	GroupId               string
	Alias                 string
	Tags                  []string
	Duration              int32
	Replica               int32
	CompleteTimeoutBlocks int
	Cid                   cid.Cid
	Rule                  string
	ExtendInfo            string
	IsUpdate              bool

	DataId    string
	OrderId   uint64
	CommitId  string
	ChunkCids []string
	TxId      string
	TxSent    bool
	Version   string
}

type OrderProposal struct {
	User       string
	Owner      string
	Provider   string
	GroupId    string
	Duration   int32
	Replica    int32
	Timeout    int32
	Alias      string
	DataId     string
	Tags       []string
	Cid        cid.Cid
	Rule       string
	IsUpdate   bool
	ExtendInfo string
}

func (o OrderProposal) Marshal() ([]byte, error) {
	return json.Marshal(o)
}

type ClientOrderProposal struct {
	Proposal        OrderProposal
	ClientSignature did.JwsSignature
}

const (
	ModelTypes = "adsf"
)

type ModelType string

const (
	ModelTypeData = ModelType("DATA")
	ModelTypeFile = ModelType("FILE")
	ModelTypeRule = ModelType("RULE")
)
