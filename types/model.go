package types

import "github.com/SaoNetwork/sao/x/sao/types"

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
	Version    string
	Content    []byte
	ExtendInfo string
}

const Type_Prefix_File = "file_"
const Type_Prefix_Model = "model_"
const Type_Prefix_Rule = "rule_"
const Type_Prefix_Schema = "schema_"
