package gql

import (
	"sao-node/node/indexer"
	"github.com/patrickmn/go-cache"
	"time"
)

type resolver struct {
	indexSvc *indexer.IndexSvc
	cache    *cache.Cache
}

func NewResolver(indexSvc *indexer.IndexSvc) *resolver {
	c := cache.New(1*time.Minute, 10*time.Minute)
	return &resolver{
		indexSvc: indexSvc,
		cache:    c,
	}
}