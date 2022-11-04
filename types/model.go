package types

import "github.com/SaoNetwork/sao/x/model/types"

type Model struct {
	DataId     string
	Alias      string
	GroupId    string
	OrderId    uint64
	Owner      string
	Tags       []string
	Cid        string
	Shards     map[string]*types.ShardMeta
	CommitId   string
	Commits    []string
	Content    []byte
	ExtendInfo string
}
