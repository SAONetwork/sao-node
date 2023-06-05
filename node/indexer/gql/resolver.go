package gql

import (
	"github.com/patrickmn/go-cache"
	"sao-node/chain"
	"sao-node/node/indexer"
	"time"
)

type resolver struct {
	indexSvc *indexer.IndexSvc
	cache    *cache.Cache
	chainSvc *chain.ChainSvc
}

func NewResolver(indexSvc *indexer.IndexSvc) *resolver {
	c := cache.New(1*time.Minute, 10*time.Minute)
	return &resolver{
		indexSvc: indexSvc,
		cache:    c,
	}
}