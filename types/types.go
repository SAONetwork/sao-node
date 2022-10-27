package types

import "github.com/ipfs/go-cid"

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
	Creator               string
	Alias                 string
	Tags                  []string
	Duration              int32
	Replica               int32
	OrderId               uint64
	CompleteTimeoutBlocks int
	Cid     cid.Cid
	Content []byte
	TxId    string
	TxSent                bool
	Rule                  string
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
