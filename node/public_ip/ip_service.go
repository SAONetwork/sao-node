package ip

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/SaoNetwork/sao-node/types"
	nodetypes "github.com/SaoNetwork/sao/x/node/types"
	logging "github.com/ipfs/go-log/v2"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	ma "github.com/multiformats/go-multiaddr"
)

var log = logging.Logger("storage")

func DoPublicIpRequest(ctx context.Context, host host.Host, nodeList []nodetypes.Node) string {
	for _, node := range nodeList {
		for _, peerId := range strings.Split(node.Peer, ",") {
			if strings.Contains(peerId, "127.0.0.1") || strings.Contains(peerId, "udp") {
				continue
			}
			log.Debug("peerId", peerId)

			a, err := ma.NewMultiaddr(peerId)
			if err != nil {
				continue
			}

			pi, err := peer.AddrInfoFromP2pAddr(a)
			if err != nil {
				continue
			}

			err = host.Connect(context.Background(), *pi)
			if err != nil {
				continue
			}

			s, err := host.NewStream(context.Background(), pi.ID, types.PublicIpProtocol)
			if err != nil {
				continue
			}

			_, err = s.Write([]byte{})
			if err != nil {
				continue
			}

			out, err := io.ReadAll(s)
			if err != nil {
				continue
			}
			return string(out)
		}
	}
	return ""
}

func HandlePublicIpRequest(s network.Stream) {
	defer s.Close()

	ma := s.Conn().RemoteMultiaddr().String()

	_, err := s.Write([]byte(strings.Split(ma, "/")[2]))
	if err != nil {
		fmt.Println(err)
	}

	if err = s.CloseWrite(); err != nil {
		fmt.Println(types.ErrCloseFileFailed, err)
		return
	}
}
