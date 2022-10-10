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
	manager := NewModelManager(config)
	require.NotNil(t, manager)
}
