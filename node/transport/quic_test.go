package transport

import (
	"context"
	"fmt"
	repo "sao-storage-node/node/repo"
	"testing"

	cid "github.com/ipfs/go-cid"
	mc "github.com/multiformats/go-multicodec"
	mh "github.com/multiformats/go-multihash"
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
	data := []byte("Hi, lao 6, how's going")
	pref := cid.Prefix{
		Version:  1,
		Codec:    uint64(mc.Raw),
		MhType:   mh.SHA2_256,
		MhLength: -1, // default length
	}
	cid, err := pref.Sum(data)
	require.NotNil(t, cid)
	require.NoError(t, err)

	result := DoQuicTransport(context.TODO(), address, peerId, data)
	require.Equal(t, cid.Bytes(), result)
}
