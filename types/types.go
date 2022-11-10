package types

import (
	did "github.com/SaoNetwork/sao-did"
	saotypes "github.com/SaoNetwork/sao/x/sao/types"
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

type ClientOrderProposal struct {
	Proposal        saotypes.Proposal
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
