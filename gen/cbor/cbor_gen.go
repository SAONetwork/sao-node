package main

import (
	"fmt"
	"os"
	"sao-node/types"

	gen "github.com/whyrusleeping/cbor-gen"
)

func main() {
	err := gen.WriteMapEncodersToFile("./types/cbor_gen.go", "types",
		// order state
		types.OrderIndex{},
		types.OrderShardInfo{},
		types.OrderInfo{},
		// shard state
		types.ShardKey{},
		types.ShardInfo{},
		types.ShardIndex{},
		// migrate state
		types.MigrateInfo{},

		types.QueryProposal{},
		types.JwsSignature{},
		types.MetadataProposalCbor{},
		types.ShardAssignReq{},
		types.ShardAssignResp{},
		types.ShardCompleteReq{},
		types.ShardCompleteResp{},
		types.ShardLoadReq{},
		types.ShardLoadResp{},
		types.ShardMigrateReq{},
		types.ShardMigrateResp{},
	)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

}
