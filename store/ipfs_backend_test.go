package store

import (
	"bytes"
	"context"
	"sao-node/utils"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSmallCid(t *testing.T) {
	b, err := NewIpfsBackend("ipfs+ma:/ip4/127.0.0.1/tcp/5001", nil)
	require.NoError(t, err)
	err = b.Open()
	require.NoError(t, err)

	data := []byte("abc")
	reader := bytes.NewReader(data)

	s, err := b.Store(context.Background(), reader)
	require.NoError(t, err)

	cid, err := utils.CalculateCid(data)
	require.NoError(t, err)
	require.Equal(t, s, cid.String())
}

func TestBigCid(t *testing.T) {
	b, err := NewIpfsBackend("ipfs+ma:/ip4/127.0.0.1/tcp/5001", nil)
	require.NoError(t, err)
	err = b.Open()
	require.NoError(t, err)

	data := []byte{}
	for i := 0; i < 1024*257; i++ {
		data = append(data, 0x01)
	}
	reader := bytes.NewReader(data)

	s, err := b.Store(context.Background(), reader)
	require.NoError(t, err)

	cid, err := utils.CalculateCid(data)
	require.NoError(t, err)
	require.Equal(t, s, cid.String())
}
