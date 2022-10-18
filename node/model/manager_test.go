package model

import (
	"sao-storage-node/node/config"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestManager(t *testing.T) {
	config := &config.Cache{
		CacheCapacity: 10,
		ContentLimit:  1024 * 1024,
	}
	manager := NewModelManager(config, nil)
	require.NotNil(t, manager)

	// model, err := manager.Create("aa", "model1", `{"xml": "3c726f6f742f3e"}`, "")
	// require.NotNil(t, model)
	// require.NoError(t, err)

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
