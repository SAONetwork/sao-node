package transport

import (
	"context"
	"sao-node/types"
	"time"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
)

func DoPingRequest(ctx context.Context, host host.Host) {
	for _, peerId := range host.Peerstore().Peers() {
		log.Debug("peerId", peerId)

		if peerId.String() == host.ID().String() {
			continue
		}

		stream, err := host.NewStream(ctx, peerId, types.ShardPingPongProtocol)
		if err != nil {
			log.Info(types.Wrap(types.ErrCreateStreamFailed, err))
			continue
		}

		defer stream.Close()
		log.Debugf("open stream to %s protocol %s.", peerId, types.ShardPingPongProtocol)

		// Set a deadline on reading from the stream so it doesn't hang
		_ = stream.SetReadDeadline(time.Now().Add(300 * time.Second))
		defer stream.SetReadDeadline(time.Time{}) // nolint

		pingpong := types.ShardPingPong{
			Local: host.ID().String(),
		}
		err = pingpong.Marshal(stream, types.FormatCbor)
		if err != nil {
			log.Error(err.Error())
			continue
		}
		if err := stream.CloseWrite(); err != nil {
			log.Error(err.Error())
			continue
		}
	}
}

func HandlePingRequest(s network.Stream) {
	defer s.Close()

	var resp types.ShardPingPong
	err := resp.Marshal(s, types.FormatCbor)
	if err != nil {
		log.Error(err.Error())
		return
	}

	if err := s.CloseWrite(); err != nil {
		log.Error(err.Error())
		return
	}
}
