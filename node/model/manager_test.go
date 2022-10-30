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
		DataId:   GenerateDataId(),
		CommitId: "888888",
	}, nil
}

func (mcs *MockCommitSvc) Pull(ctx context.Context, key string) (*storage.PullResult, error) {
	return &storage.PullResult{
		OrderId: 100,
		DataId:  GenerateDataId(),
		Content: []byte("sdafasdf"),
	}, nil
}

func (mcs *MockCommitSvc) Stop(ctx context.Context) error {
	return nil
}

func TestManager1(t *testing.T) {
	config := &config.Cache{
		EnableCache:   true,
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

	modelLoad1, err := manager.Load(context.Background(), orderMeta.Creator, orderMeta.Alias)
	require.Equal(t, model.DataId, modelLoad1.DataId)
	require.NoError(t, err)

	modelLoad2, err := manager.Load(context.Background(), orderMeta.Creator, model.DataId)
	require.Equal(t, model.Alias, modelLoad2.Alias)
	require.NoError(t, err)

	t.Log("model alias: ", model.Alias)
	t.Log("model dataId: ", model.DataId)
}

func TestManager2(t *testing.T) {
	config := &config.Cache{
		EnableCache:   true,
		CacheCapacity: 10,
		ContentLimit:  1024 * 1024,
	}

	var mockDb datastore.Read = &MockDb{}
	var mockCommitSvc storage.CommitSvcApi = &MockCommitSvc{}

	manager := NewModelManager(config, mockCommitSvc, mockDb)
	require.NotNil(t, manager)

	creator := "cosmos1080r7yvzd3ldveynuazy9ze63szn4m5tmjs60h"

	schemaOrder := types.OrderMeta{
		Creator:  creator,
		Alias:    "addresses_schema",
		Duration: 100000,
		Replica:  1,
		OrderId:  1,
		Content: []byte(`{
			"definitions": {
				"address": {
					"type": "object",
					"$id" : "cc1e76d1-e341-46eb-b3ca-102ae66d82f5",
					"properties": {
						"street_address": { "type": "string" },
						"city":           { "type": "string" },
						"state":          { "type": "string" }
					},
					"required": ["street_address", "city"]
				}
			},
			"type": "object",
			"properties": {
				"billing_address": { "$ref": "cc1e76d1-e341-46eb-b3ca-102ae66d82f5" },
				"shipping_address": { "$ref": "cc1e76d1-e341-46eb-b3ca-102ae66d82f5" }
			}
		}`),
		TxId:   "4EC45A9C04A636AA5B47A51DACCE5E64481263974B500F4DCFDD10CFDE437607",
		TxSent: true,
		Rule:   "",
	}

	schema, err := manager.Create(context.Background(), schemaOrder, types.ModelTypeData)
	require.NotNil(t, schema)
	require.NoError(t, err)

	schemaLoad1, err := manager.Load(context.Background(), creator, "addresses_schema")
	require.Equal(t, schema.Alias, schemaLoad1.Alias)
	require.NoError(t, err)

	schemaLoad2, err := manager.Load(context.Background(), creator, schema.DataId)
	require.Equal(t, schema.Alias, schemaLoad2.Alias)
	require.NoError(t, err)

	modelStr := `{
		"@context": "` + schema.DataId + `",
		"billing_address": {
			"street_address": "No. 1 Street",
			"city": "Lonton"
		},
		"shipping_address": {
			"street_address": "No. 2 Street",
			"city": "Huston",
			"state": "Texas"
		}
	}`
	modelOrder := types.OrderMeta{
		Creator:  creator,
		Alias:    "test_model",
		Duration: 100000,
		Replica:  1,
		OrderId:  1,
		Content:  []byte(modelStr),
		TxId:     "4EC45A9C04A636AA5B47A51DACCE5E64481263974B500F4DCFDD10CFDE437627",
		TxSent:   true,
		Rule:     "",
	}

	model, err := manager.Create(context.Background(), modelOrder, types.ModelTypeData)
	require.NotNil(t, model)
	require.NoError(t, err)

	modelLoad1, err := manager.Load(context.Background(), creator, "test_model")
	require.Equal(t, model.DataId, modelLoad1.DataId)
	require.NoError(t, err)

	modelLoad2, err := manager.Load(context.Background(), creator, model.DataId)
	require.Equal(t, model.Alias, modelLoad2.Alias)
	require.NoError(t, err)

	t.Log("model alias: ", model.Alias)
	t.Log("model dataId: ", model.DataId)
}
