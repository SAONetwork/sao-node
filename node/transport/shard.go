package transport

import (
	"context"
	"io"
	"sao-node/types"
	"strings"
	"time"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	ma "github.com/multiformats/go-multiaddr"

	logging "github.com/ipfs/go-log/v2"
	"github.com/libp2p/go-libp2p/core/network"
)

var log = logging.Logger("transport")

type CommonUnmarshaler interface {
	Unmarshal(io.Reader, string) error
}

type CommonMarshaler interface {
	Marshal(io.Writer, string) error
}

func HandleRequest(ctx context.Context, peerInfos string, host host.Host, protocol protocol.ID, req interface{}, resp interface{}, isForward bool) error {
	var pi *peer.AddrInfo
	for _, peerInfo := range strings.Split(peerInfos, ",") {
		if strings.Contains(peerInfo, "udp") || strings.Contains(peerInfo, "127.0.0.1") {
			continue
		}

		a, err := ma.NewMultiaddr(peerInfo)
		if err != nil {
			return types.Wrapf(types.ErrInvalidServerAddress, "peerInfo=%s", peerInfo)
		}
		pi, err = peer.AddrInfoFromP2pAddr(a)
		if err != nil {
			return types.Wrapf(types.ErrInvalidServerAddress, "a=%v", a)
		}
	}
	var stream network.Stream = nil
	var err error = nil
	if pi == nil {
		for _, peerId := range host.Peerstore().Peers() {
			log.Debug("peerId", peerId)
			if strings.Contains(peerInfos, peerId.String()) {
				stream, err = host.NewStream(ctx, peerId, protocol)
				if err != nil {
					defer stream.Close()
					return types.Wrap(types.ErrCreateStreamFailed, err)
				}
				break
			} else {
				log.Debug("not ", peerInfos)
			}
		}
		if stream == nil {
			return types.Wrap(types.ErrInvalidServerAddress, nil)
		}
	} else {
		err = host.Connect(ctx, *pi)
		if err != nil {
			return types.Wrap(types.ErrConnectFailed, err)
		}
		stream, err = host.NewStream(ctx, pi.ID, protocol)
	}

	if err != nil {
		if isForward {
			for _, peerId := range host.Peerstore().Peers() {
				relayStream, err := host.NewStream(ctx, peerId, protocol)
				if err != nil {
					log.Warn(types.Wrap(types.ErrCreateStreamFailed, err))
				}

				defer relayStream.Close()
				log.Debugf("open stream to %s protocol %s.", peerId, protocol)

				// Set a deadline on reading from the stream so it doesn't hang
				_ = relayStream.SetReadDeadline(time.Now().Add(300 * time.Second))
				defer relayStream.SetReadDeadline(time.Time{}) // nolint

				err = DoRequest(ctx, relayStream, req, resp, types.FormatCbor)
				if err != nil {
					log.Warn(types.Wrap(types.ErrCreateStreamFailed, err))
				} else {
					return nil
				}
			}
		}
		return types.Wrap(types.ErrCreateStreamFailed, err)
	}
	defer stream.Close()
	log.Debugf("open stream to %s protocol %s.", peerInfos, protocol)

	// Set a deadline on reading from the stream so it doesn't hang
	_ = stream.SetReadDeadline(time.Now().Add(300 * time.Second))
	defer stream.SetReadDeadline(time.Time{}) // nolint

	for retryTimes := 0; ; retryTimes++ {
		if err = DoRequest(ctx, stream, req, resp, types.FormatCbor); err != nil {
			if retryTimes > 2 {
				return err
			} else {
				log.Error(err)
			}
			time.Sleep(time.Second * 10)
		} else {
			break
		}
	}

	return nil
}

func DoRequest(ctx context.Context, s network.Stream, req interface{}, resp interface{}, format string) error {
	errc := make(chan error)
	go func() {
		if m, ok := req.(CommonMarshaler); ok {
			if err := m.Marshal(s, format); err != nil {
				errc <- types.Wrap(types.ErrSendRequestFailed, err)
				return
			}
			err := s.CloseWrite()
			if err != nil {
				log.Error(types.Wrap(types.ErrCloseStreamFailed, err))
				// return
			}
		} else {
			errc <- types.Wrap(types.ErrSendRequestFailed, nil)
			return
		}

		if m, ok := resp.(CommonUnmarshaler); ok {
			if err := m.Unmarshal(s, format); err != nil {
				errc <- types.Wrap(types.ErrReadResponseFailed, err)
				return
			}
		} else {
			errc <- types.Wrap(types.ErrReadResponseFailed, nil)
			return
		}

		errc <- nil
	}()

	select {
	case err := <-errc:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}
