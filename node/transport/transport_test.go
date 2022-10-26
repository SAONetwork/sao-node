package transport

import (
	"testing"
)

func TestTransport(t *testing.T) {
	// repo, err := repo.NewRepo("./testdata")
	// require.NotNil(t, repo)
	// require.NoError(t, err)
	// serverKey, err := repo.PeerId()
	// require.NotNil(t, serverKey)
	// require.NoError(t, err)
	// serverId, err := peer.IDFromPrivateKey(serverKey)
	// require.NotNil(t, serverId)
	// require.NoError(t, err)

	// err = StartTransportServer("/ip4/127.0.0.1/udp/26661", serverKey)
	// require.NoError(t, err)

	// // addrChan := make(chan ma.Multiaddr)
	// go func() {
	// 	clientKey, _, err := ic.GenerateEd25519Key(rand.Reader)
	// 	require.NotNil(t, clientKey)
	// 	require.NoError(t, err)
	// 	tr2, err := libp2pwebtransport.New(clientKey, nil, network.NullResourceManager)
	// 	require.NoError(t, err)
	// 	defer tr2.(io.Closer).Close()

	// 	fmt.Println("666")

	// 	// conn, err := tr2.Dial(context.Background(), ln.Multiaddr(), serverId)
	// 	// require.NoError(t, err)

	// 	// fmt.Println("777")
	// 	// str, err := conn.OpenStream(context.Background())
	// 	// require.NoError(t, err)
	// 	// _, err = str.Write([]byte("foobar"))
	// 	// require.NoError(t, err)
	// 	// require.NoError(t, str.Close())

	// 	// fmt.Println("888")

	// 	// data := []byte("Hi, lao 6, how's going")
	// 	// pref := cid.Prefix{
	// 	// 	Version:  1,
	// 	// 	Codec:    uint64(mc.Raw),
	// 	// 	MhType:   mh.SHA2_256,
	// 	// 	MhLength: -1, // default length
	// 	// }
	// 	// cid, err := pref.Sum(data)
	// 	// require.NotNil(t, cid)
	// 	// require.NoError(t, err)
	// 	// c := cli.DoTransport(context.TODO(), "/ip4/127.0.0.1/udp/26660/quic/webtransport/certhash/uEiBs2u9K13BQ-vX0k32I0QaN8lN95qQzoytH9wbJQDUm2w/certhash/uEiC_57QLDQ0zk0GUKMC9Vs0yC7SabtFmksf8ohZZfLQ2Pw/p2p/12D3KooWPj8B9LPYEHDPqmqsmKZirchtf3HGhUZLoZKcopVMaoJP", "12D3KooWPj8B9LPYEHDPqmqsmKZirchtf3HGhUZLoZKcopVMaoJP", data)
	// 	// require.Equal(t, cid, c)
	// 	// fmt.Println("999")

	// 	// addrChan <- conn.RemoteMultiaddr()
	// }()

	// // conn, err := ln.Accept()
	// // require.NoError(t, err)
	// // require.False(t, conn.IsClosed())
	// // str, err := conn.AcceptStream()
	// // require.NoError(t, err)
	// // data, err := io.ReadAll(str)
	// // require.NoError(t, err)
	// // require.Equal(t, "foobar", string(data))
	// // require.Equal(t, <-addrChan, conn.LocalMultiaddr())
	// // require.NoError(t, conn.Close())
	// // require.True(t, conn.IsClosed())
}
