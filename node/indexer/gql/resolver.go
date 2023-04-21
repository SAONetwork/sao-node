package gql

import (
	"sao-node/node/indexer"
)

type resolver struct {
	indexSvc *indexer.IndexSvc
}

func NewResolver(indexSvc *indexer.IndexSvc) *resolver {
	return &resolver{
		indexSvc,
	}
}
