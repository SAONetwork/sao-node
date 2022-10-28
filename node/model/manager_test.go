package model

import (
	"context"
	"sao-storage-node/node/config"
	"sao-storage-node/node/storage"
	"sao-storage-node/types"
	"testing"

	"github.com/ipfs/go-datastore"
	"github.com/ipfs/go-datastore/query"
	"github.com/stretchr/testify/require"
)

type MockDb struct {
}

type MockCommitSvc struct {
}

func (md *MockDb) Get(ctx context.Context, key datastore.Key) (value []byte, err error) {
	return nil, nil
}

func (md *MockDb) Has(ctx context.Context, key datastore.Key) (exists bool, err error) {
	return false, nil
}

func (md *MockDb) GetSize(ctx context.Context, key datastore.Key) (size int, err error) {
	return 0, nil
}

func (md *MockDb) Query(ctx context.Context, q query.Query) (query.Results, error) {
	return nil, nil
}

func (mcs *MockCommitSvc) Commit(ctx context.Context, creator string, orderMeta types.OrderMeta, content []byte) (*storage.CommitResult, error) {
	return &storage.CommitResult{
		OrderId:  100,
		DataId:   "6666666",
		CommitId: "888888",
	}, nil
}

func (mcs *MockCommitSvc) Pull(ctx context.Context, key string) (*storage.PullResult, error) {
	return &storage.PullResult{
		OrderId: 100,
		DataId:  "6666666",
		Content: []byte("sdafasdf"),
	}, nil
}

func (mcs *MockCommitSvc) Stop(ctx context.Context) error {
	return nil
}

func TestManager(t *testing.T) {
	config := &config.Cache{
		CacheCapacity: 10,
		ContentLimit:  1024 * 1024,
	}

	var mockDb datastore.Read = &MockDb{}
	var mockCommitSvc storage.CommitSvcApi = &MockCommitSvc{}

	manager := NewModelManager(config, mockCommitSvc, mockDb)
	require.NotNil(t, manager)

	orderMeta := types.OrderMeta{
		Creator:  "cosmos1080r7yvzd3ldveynuazy9ze63szn4m5tmjs60h",
		Alias:    "test_model1",
		Duration: 100000,
		Replica:  1,
		OrderId:  1,
		Content: []byte(`{
			"name": "Musk",
			"address": "Unknown",
		}`),
		TxId:   "4EC45A9C04A636AA5B47A51DACCE5E64481263974B500F4DCFDD10CFDE437607",
		TxSent: true,
		Rule:   "",
	}

	model, err := manager.Create(context.Background(), orderMeta, types.ModelTypeData)
	require.NotNil(t, model)
	require.NoError(t, err)

	// t.Logf(model.OrderId)
	// t.Logf(model.DataId)
	// t.Logf(string(model.Content))

	// model1, err1 := manager.Load("aa", "model1")
	// require.NotNil(t, model1)
	// require.NoError(t, err1)
	// t.Logf(model1.OrderId)
	// t.Logf(model1.DataId)
	// t.Logf(string(model1.Content))

	// newModel := []byte(`{"xml": "3c726f6f742f3e", "json": "1234567890abcdef"}`)
	// patch, err1 := manager.JsonpatchSvc.CreatePatch(model1.Content, newModel)
	// require.NoError(t, err1)

	// t.Logf("Patch %s", patch)

	// model1, err1 = manager.Update("aa", "model1", string(patch), "")
	// require.NotNil(t, model1)
	// require.NoError(t, err1)
	// t.Logf(model1.OrderId)
	// t.Logf(model1.DataId)
	// t.Logf(string(model1.Content))
}
