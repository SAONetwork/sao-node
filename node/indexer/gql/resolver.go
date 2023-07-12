package gql

import (
	"github.com/SaoNetwork/sao-node/node/indexer"
)

type resolver struct {
	indexSvc *indexer.IndexSvc
}

func NewResolver(indexSvc *indexer.IndexSvc) *resolver {
	return &resolver{
		indexSvc,
	}
}
