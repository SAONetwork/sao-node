package storage

import (
	"context"
	"fmt"
	"sao-node/node/transport"
	"sao-node/types"
	"strings"
	"time"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
)

type StreamStorageProtocol struct {
	host host.Host
	StorageProtocolHandler
}

func NewStreamStorageProtocol(
	host host.Host,
	handler StorageProtocolHandler,
) StreamStorageProtocol {
	ssp := StreamStorageProtocol{
		host:                   host,
		StorageProtocolHandler: handler,
	}
	host.SetStreamHandler(types.ShardAssignProtocol, ssp.handleShardAssign)
	host.SetStreamHandler(types.ShardLoadProtocol, ssp.handleShardLoad)
	host.SetStreamHandler(types.ShardMigrateProtocol, ssp.handleShardMigrate)
	return ssp
}

func (l StreamStorageProtocol) Stop(ctx context.Context) error {
	log.Info("stopping stream storage protocol")
	l.host.RemoveStreamHandler(types.ShardAssignProtocol)
	l.host.RemoveStreamHandler(types.ShardLoadProtocol)
	l.host.RemoveStreamHandler(types.ShardMigrateProtocol)
	return nil
}

func (l StreamStorageProtocol) handleShardMigrate(s network.Stream) {
	defer s.Close()

	respond := func(resp types.ShardMigrateResp) {
		err := resp.Marshal(s, types.FormatCbor)
		if err != nil {
			log.Error(err.Error())
			return
		}

		if err = s.CloseWrite(); err != nil {
			log.Error(err.Error())
			return
		}
	}

	// Set a deadline on reading from the stream so it doesn't hang
	_ = s.SetReadDeadline(time.Now().Add(30 * time.Second))
	defer s.SetReadDeadline(time.Time{}) // nolint

	var req types.ShardMigrateReq
	err := req.Unmarshal(s, types.FormatCbor)
	if err != nil {
		respond(types.ShardMigrateResp{
			Code:    types.ErrorCodeInternalErr,
			Message: fmt.Sprintf("failed to unmarshal request: %v", err),
		})
		return
	}
	respond(l.HandleShardMigrate(req))
}

func (l StreamStorageProtocol) handleShardLoad(s network.Stream) {
	defer s.Close()

	respond := func(resp types.ShardLoadResp) {
		err := resp.Marshal(s, types.FormatCbor)
		if err != nil {
			log.Error(err.Error())
			return
		}

		if err = s.CloseWrite(); err != nil {
			log.Error(err.Error())
			return
		}
	}

	// Set a deadline on reading from the stream so it doesn't hang
	_ = s.SetReadDeadline(time.Now().Add(30 * time.Second))
	defer s.SetReadDeadline(time.Time{}) // nolint

	var req types.ShardLoadReq
	err := req.Unmarshal(s, types.FormatCbor)
	if err != nil {
		respond(types.ShardLoadResp{
			Code:       types.ErrorCodeInvalidRequest,
			Message:    fmt.Sprintf("failed to unmarshal request: %v", err),
			OrderId:    req.OrderId,
			Cid:        req.Cid,
			RequestId:  req.RequestId,
			ResponseId: time.Now().UnixMilli(),
		})
		return
	}
	peerInfo := string(s.Conn().RemotePeer())
	log.Debug("check peer: %v<->%v", req.Proposal.Proposal.Gateway, peerInfo)
	if !strings.Contains(req.Proposal.Proposal.Gateway, peerInfo) {
		respond(types.ShardLoadResp{
			Code:       types.ErrorCodeInternalErr,
			Message:    fmt.Sprintf("invalid query, unexpect gateway:%s, should be %s", peerInfo, req.Proposal.Proposal.Gateway),
			OrderId:    req.OrderId,
			Cid:        req.Cid,
			RequestId:  req.RequestId,
			ResponseId: time.Now().UnixMilli(),
		})
		return
	}
	respond(l.HandleShardLoad(req))
}

func (l StreamStorageProtocol) handleShardAssign(s network.Stream) {
	defer s.Close()

	respond := func(resp types.ShardAssignResp) {
		err := resp.Marshal(s, types.FormatCbor)
		if err != nil {
			log.Error(err.Error())
			return
		}

		if err = s.CloseWrite(); err != nil {
			log.Error(err.Error())
			return
		}
	}

	// Set a deadline on reading from the stream so it doesn't hang
	_ = s.SetReadDeadline(time.Now().Add(30 * time.Second))
	defer s.SetReadDeadline(time.Time{}) // nolint

	var req types.ShardAssignReq
	err := req.Unmarshal(s, types.FormatCbor)
	if err != nil {
		respond(types.ShardAssignResp{
			Code:    types.ErrorCodeInvalidRequest,
			Message: fmt.Sprintf("failed to unmarshal request: %v", err),
		})
	}
	respond(l.HandleShardAssign(req))
}

func (l StreamStorageProtocol) RequestShardMigrate(
	ctx context.Context,
	req types.ShardMigrateReq,
	peer string,
) types.ShardMigrateResp {
	resp := types.ShardMigrateResp{}
	err := transport.HandleRequest(ctx, peer, l.host, types.ShardMigrateProtocol, &req, &resp)
	if err != nil {
		resp = types.ShardMigrateResp{
			Code:    types.ErrorCodeInternalErr,
			Message: fmt.Sprint("transport migrate request error: %v", err),
		}
	}
	return resp
}

func (l StreamStorageProtocol) RequestShardComplete(ctx context.Context, req types.ShardCompleteReq, peer string) types.ShardCompleteResp {
	resp := types.ShardCompleteResp{}
	err := transport.HandleRequest(
		ctx,
		peer,
		l.host,
		types.ShardCompleteProtocol,
		&req,
		&resp,
	)
	if err != nil {
		resp = types.ShardCompleteResp{
			Code:        types.ErrorCodeInternalErr,
			Message:     fmt.Sprintf("transport complete request error: %v", err),
			Recoverable: true,
		}
	}
	return resp
}

func (l StreamStorageProtocol) RequestShardStore(ctx context.Context, req types.ShardLoadReq, peer string) types.ShardLoadResp {
	resp := types.ShardLoadResp{}
	err := transport.HandleRequest(
		ctx,
		peer,
		l.host,
		types.ShardStoreProtocol,
		&req,
		&resp,
	)
	if err != nil {
		resp = types.ShardLoadResp{
			Code:       types.ErrorCodeInternalErr,
			Message:    fmt.Sprintf("transport complete request error: %v", err),
			OrderId:    req.OrderId,
			Cid:        req.Cid,
			RequestId:  req.RequestId,
			ResponseId: time.Now().UnixMilli(),
		}
	}
	return resp
}
