package gateway

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	ip "github.com/SaoNetwork/sao-node/node/public_ip"
	"github.com/SaoNetwork/sao-node/node/transport"
	"github.com/SaoNetwork/sao-node/types"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
)

type StreamGatewayProtocol struct {
	ctx  context.Context
	host host.Host
	RH   *transport.RpcHandler
	GatewayProtocolHandler
	LocalGatewayProtocol
}

func NewStreamGatewayProtocol(ctx context.Context, host host.Host, handler GatewayProtocolHandler, local LocalGatewayProtocol, rh *transport.RpcHandler) StreamGatewayProtocol {
	sgp := StreamGatewayProtocol{
		ctx:                    ctx,
		host:                   host,
		GatewayProtocolHandler: handler,
		LocalGatewayProtocol:   local,
		RH:                     rh,
	}
	host.SetStreamHandler(types.ShardStoreProtocol, sgp.handleShardStoreStream)
	host.SetStreamHandler(types.ShardCompleteProtocol, sgp.handleShardCompleteStream)
	host.SetStreamHandler(types.ShardLoadProtocol, sgp.handleRelayStream)
	host.SetStreamHandler(types.ShardPingPongProtocol, transport.HandlePingRequest)
	host.SetStreamHandler(types.RpcProtocol, sgp.HandleRPCRequest)
	host.SetStreamHandler(types.PublicIpProtocol, ip.HandlePublicIpRequest)
	return sgp
}

func (l StreamGatewayProtocol) Stop(ctx context.Context) error {
	log.Info("stopping stream gateway protocol ...")
	l.host.RemoveStreamHandler(types.ShardStoreProtocol)
	l.host.RemoveStreamHandler(types.ShardCompleteProtocol)
	return nil
}
func (l StreamGatewayProtocol) HandleRPCRequest(s network.Stream) {
	defer s.Close()

	// Set a deadline on reading from the stream so it doesn’t hang
	_ = s.SetReadDeadline(time.Now().Add(30 * time.Second))
	defer s.SetReadDeadline(time.Time{}) // nolint

	var req types.RpcReq
	var resp = types.RpcResp{}

	buf := &bytes.Buffer{}
	buf.ReadFrom(s)
	err := json.Unmarshal(buf.Bytes(), &req)
	if err == nil {
		log.Info("Got rpc request: ", req.Method)

		var result string
		var err error
		switch req.Method {
		case "Sao.Upload":
			req.Params = append(req.Params, filepath.Join(l.RH.StagingPath, s.Conn().RemotePeer().String()))
			result, err = l.RH.Upload(req.Params)
		case "Sao.ModelCreate":
			result, err = l.RH.Create(req.Params)
		case "Sao.ModelLoad":
			result, err = l.RH.Load(req.Params)
		case "Sao.ModelUpdate":
			result, err = l.RH.Update(req.Params)
		default:
			resp.Error = "N/a"
		}
		if err != nil {
			resp.Error = err.Error()
		} else {
			resp.Data = result
		}

	} else {
		resp.Error = err.Error()
	}

	bytes, err := json.Marshal(resp)
	if err != nil {
		log.Error(err.Error())
		return
	}

	if _, err := s.Write(bytes); err != nil {
		log.Error(err.Error())
		return
	}

	if err := s.CloseWrite(); err != nil {
		log.Error(err.Error())
		return
	}

	log.Info("Sent rpc response: ", resp)
}

func (l StreamGatewayProtocol) handleShardStoreStream(s network.Stream) {
	log.Infof("handling %s ...", types.ShardStoreProtocol)
	defer s.Close()

	respond := func(resp types.ShardLoadResp) {
		err := resp.Marshal(s, types.FormatCbor)
		if err != nil {
			log.Error(types.Wrap(types.ErrMarshalFailed, err))
			return
		}

		if err = s.CloseWrite(); err != nil {
			log.Error(types.Wrap(types.ErrCloseFileFailed, err))
			return
		}
	}

	// Set a deadline on reading from the stream so it doesn't hang
	_ = s.SetReadDeadline(time.Now().Add(30 * time.Second))
	defer s.SetReadDeadline(time.Time{}) // nolint

	var req types.ShardLoadReq
	err := req.Unmarshal(s, types.FormatCbor)
	if err != nil {
		log.Error(types.Wrap(types.ErrUnMarshalFailed, err))
		respond(types.ShardLoadResp{
			Code:    types.ErrorCodeInvalidRequest,
			Message: fmt.Sprintf("failed to unmarshal request: %v", err),
		})
		return
	}
	log.Debugf("receive ShardLoadReq: orderId=%d cid=%v requestId=%d", req.OrderId, req.Cid, req.RequestId)

	respond(l.HandleShardStore(req))
}

func (l StreamGatewayProtocol) handleShardCompleteStream(s network.Stream) {
	log.Infof("handling %s ...", types.ShardCompleteProtocol)
	defer s.Close()

	respond := func(resp types.ShardCompleteResp) {
		err := resp.Marshal(s, types.FormatCbor)
		if err != nil {
			log.Error(types.Wrap(types.ErrMarshalFailed, err))
			return
		}

		if err = s.CloseWrite(); err != nil {
			log.Error(types.Wrap(types.ErrCloseFileFailed, err))
			return
		}
	}

	// Set a deadline on reading from the stream so it doesn't hang
	_ = s.SetReadDeadline(time.Now().Add(30 * time.Second))
	defer s.SetReadDeadline(time.Time{}) // nolint

	var req types.ShardCompleteReq
	err := req.Unmarshal(s, types.FormatCbor)
	if err != nil {
		log.Error(types.Wrap(types.ErrUnMarshalFailed, err))
		respond(types.ShardCompleteResp{
			Code:    types.ErrorCodeInvalidRequest,
			Message: fmt.Sprintf("failed to unmarshal request: %v", err),
		})
		return
	}

	respond(l.HandleShardComplete(req))
}

func (l StreamGatewayProtocol) handleRelayStream(s network.Stream) {
	log.Infof("handling relay %s ...", types.ShardLoadProtocol)
	defer s.Close()

	respond := func(resp types.ShardLoadResp) {
		err := resp.Marshal(s, types.FormatCbor)
		if err != nil {
			log.Error(types.Wrap(types.ErrMarshalFailed, err))
			return
		}

		if err = s.CloseWrite(); err != nil {
			log.Error(types.Wrap(types.ErrCloseFileFailed, err))
			return
		}
	}

	// Set a deadline on reading from the stream so it doesn't hang
	_ = s.SetReadDeadline(time.Now().Add(30 * time.Second))
	defer s.SetReadDeadline(time.Time{}) // nolint

	var req types.ShardLoadReq
	err := req.Unmarshal(s, types.FormatCbor)
	if err != nil {
		log.Error(types.Wrap(types.ErrUnMarshalFailed, err))
		respond(types.ShardLoadResp{
			Code:    types.ErrorCodeInvalidRequest,
			Message: fmt.Sprintf("failed to unmarshal request: %v", err),
		})
		return
	}
	log.Debugf("receive Relay ShardLoadReq: orderId=%d cid=%v requestId=%d", req.OrderId, req.Cid, req.RequestId)

	if strings.Contains(req.RelayProposal.Proposal.TargetPeerInfo, l.host.ID().String()) {
		// should not happen
		respond(l.LocalGatewayProtocol.RequestShardLoad(l.ctx, req, req.RelayProposal.Proposal.TargetPeerInfo, false))
	} else {
		for _, peer := range l.host.Peerstore().Peers() {
			if strings.Contains(req.RelayProposal.Proposal.TargetPeerInfo, peer.String()) {
				respond(l.RequestShardLoad(l.ctx, req, req.RelayProposal.Proposal.TargetPeerInfo, false))
				break
			}
		}
	}
}

func (l StreamGatewayProtocol) RequestShardAssign(ctx context.Context, req types.ShardAssignReq, peer string) types.ShardAssignResp {
	var resp types.ShardAssignResp
	err := transport.HandleRequest(
		ctx,
		peer,
		l.host,
		types.ShardAssignProtocol,
		&req,
		&resp,
		false,
	)
	if err != nil {
		resp = types.ShardAssignResp{
			Code:    types.ErrorCodeInternalErr,
			Message: fmt.Sprintf("transport assign request error: %v", err),
		}
	}
	return resp
}

func (l StreamGatewayProtocol) RequestShardLoad(ctx context.Context, req types.ShardLoadReq, peer string, isForward bool) types.ShardLoadResp {
	var resp types.ShardLoadResp
	err := transport.HandleRequest(
		ctx,
		peer,
		l.host,
		types.ShardLoadProtocol,
		&req,
		&resp,
		isForward,
	)
	if err != nil {
		resp = types.ShardLoadResp{
			Code:       types.ErrorCodeInternalErr,
			Message:    fmt.Sprintf("transport assign request error: %v", err),
			OrderId:    req.OrderId,
			Cid:        req.Cid,
			Content:    nil,
			RequestId:  req.RequestId,
			ResponseId: time.Now().UnixMilli(),
		}
	}
	return resp
}

func (l StreamGatewayProtocol) GetPeers(_ context.Context) string {
	return l.host.Peerstore().Peers().String()
}
