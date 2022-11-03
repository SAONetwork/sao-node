package types

import "github.com/SaoNetwork/sao/x/sao/types"

type Model struct {
	DataId     string
	Alias      string
	GroupId    string
	OrderId    uint64
	Creator    string
	Tags       []string
	Cid        string
	ChunkCids  []string
	Shards     map[string]*types.Shard
	CommitId   string
	Commits    []string
	Content    []byte
	ExtendInfo string
}
