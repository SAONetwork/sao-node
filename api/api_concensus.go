package api

import "context"

type SubscribeQuery struct {
	Query string
}

type SubscribeResult struct {
	Query string
}

type ConcensusNodeApi interface {
	Subscribe(ctx context.Context, query SubscribeQuery) (<-chan []*SubscribeResult, error)
}
