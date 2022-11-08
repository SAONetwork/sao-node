package model

import (
	"context"
	"sao-storage-node/node/config"
	"sao-storage-node/node/gateway"
	"sao-storage-node/types"
	"testing"

	"github.com/stretchr/testify/require"
)

type MockGatewaySvc struct {
}

func (mcs *MockGatewaySvc) CommitModel(ctx context.Context, creator string, orderMeta types.OrderMeta, content []byte) (*gateway.CommitResult, error) {
	return &gateway.CommitResult{
		OrderId: 100,
		DataId:  orderMeta.DataId,
		Commit:  orderMeta.DataId,
	}, nil
}

func (mcs *MockGatewaySvc) QueryMeta(ctx context.Context, account string, key string, group string, height int64) (*types.Model, error) {
	return &types.Model{
		OrderId:  100,
		DataId:   group,
		CommitId: group,
		Alias:    key,
		Cid:      key,
	}, nil
}

func (os *MockGatewaySvc) FetchContent(ctx context.Context, meta *types.Model) (*gateway.FetchResult, error) {
	return &gateway.FetchResult{
		Cid:     "123",
		Content: make([]byte, 0),
	}, nil
}

func (cs *MockGatewaySvc) Stop(ctx context.Context) error {
	return nil
}

func TestManager1(t *testing.T) {
	config := &config.Cache{
		EnableCache:   true,
		CacheCapacity: 10,
		ContentLimit:  1024 * 1024,
	}

	var mockGatewaySvc gateway.GatewaySvcApi = &MockGatewaySvc{}

	manager := NewModelManager(config, mockGatewaySvc)
	require.NotNil(t, manager)

	orderMeta := types.OrderMeta{
		Owner:    "cosmos1080r7yvzd3ldveynuazy9ze63szn4m5tmjs60h",
		DataId:   "5e1f67df-0a22-4798-a9dc-a9d9a74722a3",
		GroupId:  "5e1f67df-0a22-4798-a9dc-a9d9a74722a3",
		Alias:    "test_model_1",
		Duration: 100000,
		Replica:  1,
		OrderId:  1,
		TxId:     "4EC45A9C04A636AA5B47A51DACCE5E64481263974B500F4DCFDD10CFDE437607",
		TxSent:   true,
		Rule:     "",
	}
	content := []byte(`{
		"name": "Musk",
		"address": "Unknown",
	}`)

	model, err := manager.Create(context.Background(), orderMeta, content)
	require.NoError(t, err)
	require.NotNil(t, model)

	modelLoad1, err := manager.Load(context.Background(), orderMeta)
	require.Equal(t, model.DataId, modelLoad1.DataId)
	require.NoError(t, err)

	modelLoad2, err := manager.Load(context.Background(), orderMeta)
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

	var mockGatewaySvc gateway.GatewaySvcApi = &MockGatewaySvc{}

	manager := NewModelManager(config, mockGatewaySvc)
	require.NotNil(t, manager)

	creator := "cosmos1080r7yvzd3ldveynuazy9ze63szn4m5tmjs60h"

	schemaOrder1 := types.OrderMeta{
		Owner:    creator,
		DataId:   "37743e1c-b4d0-42f6-9fee-89592425c1fb",
		GroupId:  "37743e1c-b4d0-42f6-9fee-89592425c1fb",
		Alias:    "addresses_schema_1",
		Duration: 100000,
		Replica:  1,
		OrderId:  1,
		TxId:     "4EC45A9C04A636AA5B47A51DACCE5E64481263974B500F4DCFDD10CFDE437607",
		TxSent:   true,
		Rule:     "",
	}
	content1 := []byte(`{
		"definitions": {
			"address1": {
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
			"billing_address": { "$ref": "cc1e76d1-e341-46eb-b3ca-102ae66d82f5" }
		}
	}`)

	schema1, err1 := manager.Create(context.Background(), schemaOrder1, content1)
	require.NotNil(t, schema1)
	require.NoError(t, err1)

	schemaLoad1, err := manager.Load(context.Background(), schemaOrder1)
	require.Equal(t, schema1.Alias, schemaLoad1.Alias)
	require.NoError(t, err)

	schemaOrder2 := types.OrderMeta{
		Owner:    creator,
		DataId:   "ba8c31a3-e1ff-408d-858a-bd8da6b23b90",
		GroupId:  "ba8c31a3-e1ff-408d-858a-bd8da6b23b90",
		Alias:    "addresses_schema_2",
		Duration: 100000,
		Replica:  1,
		OrderId:  1,
		TxId:     "4EC45A9C04A636AA5B47A51DACCE5E64481263974B500F4DCFDD10CFDE437627",
		TxSent:   true,
		Rule:     "",
	}
	content2 := []byte(`{
		"definitions": {
			"address": {
				"type": "object",
				"$id" : "cc1e76d1-e341-46eb-b3ca-102ae66d82f5",
				"properties": {
					"street_address": { "type": "string" },
					"city":           { "type": "string" },
					"state":          { "type": "string" }
				},
				"required": ["street_address", "state"]
			}
		},
		"type": "object",
		"properties": {
			"shipping_address": { "$ref": "cc1e76d1-e341-46eb-b3ca-102ae66d82f5" }
		}
	}`)

	schema2, err2 := manager.Create(context.Background(), schemaOrder2, content2)
	require.NotNil(t, schema2)
	require.NoError(t, err2)

	schemaLoad2, err2 := manager.Load(context.Background(), schemaOrder2)
	require.Equal(t, schema2.Alias, schemaLoad2.Alias)
	require.NoError(t, err2)

	modelStr := `{
		"@context": ["` + schema1.DataId + `", "` + schema2.DataId + `"],
		"billing_address": {
			"street_address": "No. 1 Street",
			"city": "Lonton"
		},
		"shipping_address": {
			"street_address": "No. 2 Street",
			"state": "Texas"
		}
	}`
	modelOrder := types.OrderMeta{
		Owner:    creator,
		DataId:   "94648406-8aeb-48ff-94c0-cc7f35e17978",
		GroupId:  "94648406-8aeb-48ff-94c0-cc7f35e17978",
		Alias:    "test_model",
		Duration: 100000,
		Replica:  1,
		OrderId:  1,
		TxId:     "4EC45A9C04A636AA5B47A51DACCE5E64481263974B500F4DCFDD10CFDE437627",
		TxSent:   true,
		Rule:     "",
	}

	model, err := manager.Create(context.Background(), modelOrder, []byte(modelStr))
	require.NoError(t, err)
	require.NotNil(t, model)

	modelLoad1, err := manager.Load(context.Background(), modelOrder)
	require.Equal(t, model.DataId, modelLoad1.DataId)
	require.Equal(t, model.Alias, modelLoad1.Alias)
	require.NoError(t, err)

	t.Log("model alias: ", model.Alias)
	t.Log("model dataId: ", model.DataId)
}
