package node

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/libp2p/go-libp2p/core/network"
	"io"
)

const (
	ShardStoreProtocol = "/sao/store/shard/1.0"
)

// TODO: store node should sign the request.
type ShardStoreReq struct {
	OrderId uint64
	Cid     string
}

func (f *ShardStoreReq) Unmarshal(r io.Reader, format string) (err error) {
	if format == "json" {
		buf := &bytes.Buffer{}
		buf.ReadFrom(r)
		err = json.Unmarshal(buf.Bytes(), f)
	} else {
		// TODO: CBOR marshal
	}
	return nil
}

func (f *ShardStoreReq) Marshal(w io.Writer, format string) error {
	if format == "json" {
		bytes, err := json.Marshal(f)
		if err != nil {
			return err
		}
		w.Write(bytes)
		return nil
	} else {
		// TODO: CBOR marshal
	}
	return nil
}

type ShardStoreResp struct {
	OrderId uint64
	Cid     string
	Content []byte
}

func (f *ShardStoreResp) Marshal(w io.Writer, format string) error {
	if format == "cbor" {
		// TODO:
	} else {
		bytes, err := json.Marshal(f)
		if err != nil {
			return err
		}
		w.Write(bytes)
	}
	return nil
}
func (f *ShardStoreResp) Unmarshal(r io.Reader, format string) (err error) {
	if format == "json" {
		buf := &bytes.Buffer{}
		buf.ReadFrom(r)
		err = json.Unmarshal(buf.Bytes(), f)
	} else {
		// TODO: CBOR marshal
	}
	return nil
}

type CommonUnmarshaler interface {
	Unmarshal(io.Reader, string) error
}

type CommonMarshaler interface {
	Marshal(io.Writer, string) error
}

func DoRpc(ctx context.Context, s network.Stream, req interface{}, resp interface{}, format string) error {
	errc := make(chan error)
	go func() {
		if m, ok := req.(CommonMarshaler); ok {
			if err := m.Marshal(s, format); err != nil {
				errc <- fmt.Errorf("failed to send request: %w", err)
				return
			}
		} else {
			errc <- fmt.Errorf("failed to send request")
			return
		}

		if m, ok := resp.(CommonUnmarshaler); ok {
			if err := m.Unmarshal(s, format); err != nil {
				errc <- fmt.Errorf("failed to read response: %w", err)
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
