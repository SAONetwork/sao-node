package main

import (
	"fmt"
	"os"
	"sao-node/types"

	gen "github.com/whyrusleeping/cbor-gen"
)

func main() {
	err := gen.WriteMapEncodersToFile("./types/cbor_gen.go", "types",
		// share expire
		types.ShardExpireKey{},
		types.ShardExpireIndex{},
		types.ShardExpireInfo{},
		// order state
		types.OrderKey{},
		types.OrderIndex{},
		types.OrderShardInfo{},
		types.OrderInfo{},
		// shard state
		types.ShardKey{},
		types.ShardInfo{},
		types.ShardIndex{},
		// migrate state
		types.MigrateKey{},
		types.MigrateInfo{},
		types.MigrateIndex{},

		types.QueryProposal{},
		types.RelayProposal{},
		types.JwsSignature{},
		types.MetadataProposalCbor{},
		types.RelayProposalCbor{},
		types.ShardAssignReq{},
		types.ShardAssignResp{},
		types.ShardCompleteReq{},
		types.ShardCompleteResp{},
		types.ShardLoadReq{},
		types.ShardLoadResp{},
		types.ShardMigrateReq{},
		types.ShardMigrateResp{},
		types.ShardPingPong{},
	)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

}
