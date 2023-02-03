package main

import (
	"fmt"
	"os"
	"sao-node/types"

	gen "github.com/whyrusleeping/cbor-gen"
)

func main() {
	err := gen.WriteMapEncodersToFile("./types/cbor_gen.go", "types",
		types.OrderStats{},
		types.ShardInfo{},
		types.OrderInfo{},
		types.QueryProposal{},
		types.JwsSignature{},
		types.MetadataProposalCbor{},
		types.ShardAssignReq{},
		types.ShardAssignResp{},
		types.ShardCompleteReq{},
		types.ShardCompleteResp{},
		types.ShardLoadReq{},
		types.ShardLoadResp{},
	)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

}
