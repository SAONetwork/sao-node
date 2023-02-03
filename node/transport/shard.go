package transport

import (
	"context"
	"fmt"
	"io"
	"sao-node/types"
	"strings"
	"time"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	ma "github.com/multiformats/go-multiaddr"
	"golang.org/x/xerrors"

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

func HandleRequest(ctx context.Context, peerInfos string, host host.Host, protocol protocol.ID, req interface{}, resp interface{}) error {
	var pi *peer.AddrInfo
	for _, peerInfo := range strings.Split(peerInfos, ",") {
		if strings.Contains(peerInfo, "udp") || strings.Contains(peerInfo, "127.0.0.1") {
			continue
		}

		a, err := ma.NewMultiaddr(peerInfo)
		if err != nil {
			return err
		}
		pi, err = peer.AddrInfoFromP2pAddr(a)
		if err != nil {
			return err
		}
	}
	if pi == nil {
		return xerrors.Errorf("failed to get peer info")
	}

	err := host.Connect(ctx, *pi)
	if err != nil {
		return err
	}
	stream, err := host.NewStream(ctx, pi.ID, protocol)
	if err != nil {
		return err
	}
	defer stream.Close()
	log.Debugf("open stream to %s protocol %s.", pi.ID, protocol)

	// Set a deadline on reading from the stream so it doesn't hang
	_ = stream.SetReadDeadline(time.Now().Add(300 * time.Second))
	defer stream.SetReadDeadline(time.Time{}) // nolint

	if err = DoRequest(ctx, stream, req, resp, types.FormatCbor); err != nil {
		// TODO: handle error
		log.Error(err)
		return err
	}
	return nil
}

func DoRequest(ctx context.Context, s network.Stream, req interface{}, resp interface{}, format string) error {
	errc := make(chan error)
	go func() {
		if m, ok := req.(CommonMarshaler); ok {
			if err := m.Marshal(s, format); err != nil {
				errc <- fmt.Errorf("failed to send request: %v", err)
				return
			}
			err := s.CloseWrite()
			if err != nil {
				log.Error(err.Error())
			}
		} else {
			errc <- fmt.Errorf("failed to send request")
			return
		}

		if m, ok := resp.(CommonUnmarshaler); ok {
			if err := m.Unmarshal(s, format); err != nil {
				errc <- fmt.Errorf("failed to read response: %v", err)
				return
			}
		} else {
			errc <- fmt.Errorf("failed to read response")
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
