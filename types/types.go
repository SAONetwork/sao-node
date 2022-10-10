package types

import "github.com/ipfs/go-cid"

type CommitHeader struct {
	Controllers []string
}

type GenesisCommitHeader struct {
	CommitHeader
	DataType string // model, record
}

type DataCommit struct {
	Content any
}

type GenesisCommit struct {
	DataCommit
	Header GenesisCommitHeader
}

type RawCommit struct {
	DataCommit
	Header CommitHeader
	Id     cid.Cid
	Prev   cid.Cid
}

type OrderMeta struct {
	Creator  string
	OrderId  string
	TxId     string
	Duration int
	Replica  int
}
