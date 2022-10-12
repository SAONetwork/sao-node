package transport

import (
	"context"
	"fmt"
	repo "sao-storage-node/node/repo"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTransport(t *testing.T) {
	repo, err := repo.NewRepo("./testdata")
	require.NotNil(t, repo)
	require.NoError(t, err)
	port := 26659
	peerId := "12D3KooWGxJNcMSuzaEiHmxGLYBmFJ7rG5ttnwMdRSX6ySBs1vrR"
	address := fmt.Sprintf("/ip4/0.0.0.0/udp/%d/quic", port)
	ServeQuicTransport(address, repo)
	result := DoQuicTransport(context.TODO(), address, peerId, []byte("Hi, lao 6, how's going"))
	require.Equal(t, []byte("OK"), result)
}
