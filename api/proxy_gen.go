// Code generated by github.com/filecoin-project/lotus/gen/api. DO NOT EDIT.

package api

import (
	"context"
	apitypes "sao-node/api/types"
	"sao-node/types"

	"github.com/filecoin-project/go-jsonrpc/auth"
	"golang.org/x/xerrors"
)

var ErrNotSupported = xerrors.New("method not supported")

type GatewayApiStruct struct {
	Internal struct {
		AuthNew func(p0 context.Context, p1 []auth.Permission) ([]byte, error) `perm:"admin"`

		AuthVerify func(p0 context.Context, p1 string) ([]auth.Permission, error) `perm:"none"`

		Create func(p0 context.Context, p1 *types.MetadataProposal, p2 *types.OrderStoreProposal, p3 uint64, p4 []byte) (apitypes.CreateResp, error) `perm:"write"`

		CreateFile func(p0 context.Context, p1 *types.MetadataProposal, p2 *types.OrderStoreProposal, p3 uint64) (apitypes.CreateResp, error) `perm:"write"`

		Delete func(p0 context.Context, p1 *types.OrderTerminateProposal) (apitypes.DeleteResp, error) `perm:"write"`

		GenerateToken func(p0 context.Context, p1 string) (apitypes.GenerateTokenResp, error) `perm:"read"`

		GetHttpUrl func(p0 context.Context, p1 string) (apitypes.GetUrlResp, error) `perm:"read"`

		GetIpfsUrl func(p0 context.Context, p1 string) (apitypes.GetUrlResp, error) `perm:"read"`

		GetPeerInfo func(p0 context.Context) (apitypes.GetPeerInfoResp, error) `perm:"read"`

		Load func(p0 context.Context, p1 *types.MetadataProposal) (apitypes.LoadResp, error) `perm:"read"`

		NetPeers func(p0 context.Context) ([]types.PeerInfo, error) `perm:"read"`

		NodeAddress func(p0 context.Context) (string, error) `perm:"read"`

		ShowCommits func(p0 context.Context, p1 *types.MetadataProposal) (apitypes.ShowCommitsResp, error) `perm:"read"`

		Test func(p0 context.Context, p1 string) (string, error) `perm:"none"`

		Update func(p0 context.Context, p1 *types.MetadataProposal, p2 *types.OrderStoreProposal, p3 uint64, p4 []byte) (apitypes.UpdateResp, error) `perm:"write"`
	}
}

type GatewayApiStub struct {
}

func (s *GatewayApiStruct) AuthNew(p0 context.Context, p1 []auth.Permission) ([]byte, error) {
	if s.Internal.AuthNew == nil {
		return *new([]byte), ErrNotSupported
	}
	return s.Internal.AuthNew(p0, p1)
}

func (s *GatewayApiStub) AuthNew(p0 context.Context, p1 []auth.Permission) ([]byte, error) {
	return *new([]byte), ErrNotSupported
}

func (s *GatewayApiStruct) AuthVerify(p0 context.Context, p1 string) ([]auth.Permission, error) {
	if s.Internal.AuthVerify == nil {
		return *new([]auth.Permission), ErrNotSupported
	}
	return s.Internal.AuthVerify(p0, p1)
}

func (s *GatewayApiStub) AuthVerify(p0 context.Context, p1 string) ([]auth.Permission, error) {
	return *new([]auth.Permission), ErrNotSupported
}

func (s *GatewayApiStruct) Create(p0 context.Context, p1 *types.MetadataProposal, p2 *types.OrderStoreProposal, p3 uint64, p4 []byte) (apitypes.CreateResp, error) {
	if s.Internal.Create == nil {
		return *new(apitypes.CreateResp), ErrNotSupported
	}
	return s.Internal.Create(p0, p1, p2, p3, p4)
}

func (s *GatewayApiStub) Create(p0 context.Context, p1 *types.MetadataProposal, p2 *types.OrderStoreProposal, p3 uint64, p4 []byte) (apitypes.CreateResp, error) {
	return *new(apitypes.CreateResp), ErrNotSupported
}

func (s *GatewayApiStruct) CreateFile(p0 context.Context, p1 *types.MetadataProposal, p2 *types.OrderStoreProposal, p3 uint64) (apitypes.CreateResp, error) {
	if s.Internal.CreateFile == nil {
		return *new(apitypes.CreateResp), ErrNotSupported
	}
	return s.Internal.CreateFile(p0, p1, p2, p3)
}

func (s *GatewayApiStub) CreateFile(p0 context.Context, p1 *types.MetadataProposal, p2 *types.OrderStoreProposal, p3 uint64) (apitypes.CreateResp, error) {
	return *new(apitypes.CreateResp), ErrNotSupported
}

func (s *GatewayApiStruct) Delete(p0 context.Context, p1 *types.OrderTerminateProposal) (apitypes.DeleteResp, error) {
	if s.Internal.Delete == nil {
		return *new(apitypes.DeleteResp), ErrNotSupported
	}
	return s.Internal.Delete(p0, p1)
}

func (s *GatewayApiStub) Delete(p0 context.Context, p1 *types.OrderTerminateProposal) (apitypes.DeleteResp, error) {
	return *new(apitypes.DeleteResp), ErrNotSupported
}

func (s *GatewayApiStruct) GenerateToken(p0 context.Context, p1 string) (apitypes.GenerateTokenResp, error) {
	if s.Internal.GenerateToken == nil {
		return *new(apitypes.GenerateTokenResp), ErrNotSupported
	}
	return s.Internal.GenerateToken(p0, p1)
}

func (s *GatewayApiStub) GenerateToken(p0 context.Context, p1 string) (apitypes.GenerateTokenResp, error) {
	return *new(apitypes.GenerateTokenResp), ErrNotSupported
}

func (s *GatewayApiStruct) GetHttpUrl(p0 context.Context, p1 string) (apitypes.GetUrlResp, error) {
	if s.Internal.GetHttpUrl == nil {
		return *new(apitypes.GetUrlResp), ErrNotSupported
	}
	return s.Internal.GetHttpUrl(p0, p1)
}

func (s *GatewayApiStub) GetHttpUrl(p0 context.Context, p1 string) (apitypes.GetUrlResp, error) {
	return *new(apitypes.GetUrlResp), ErrNotSupported
}

func (s *GatewayApiStruct) GetIpfsUrl(p0 context.Context, p1 string) (apitypes.GetUrlResp, error) {
	if s.Internal.GetIpfsUrl == nil {
		return *new(apitypes.GetUrlResp), ErrNotSupported
	}
	return s.Internal.GetIpfsUrl(p0, p1)
}

func (s *GatewayApiStub) GetIpfsUrl(p0 context.Context, p1 string) (apitypes.GetUrlResp, error) {
	return *new(apitypes.GetUrlResp), ErrNotSupported
}

func (s *GatewayApiStruct) GetPeerInfo(p0 context.Context) (apitypes.GetPeerInfoResp, error) {
	if s.Internal.GetPeerInfo == nil {
		return *new(apitypes.GetPeerInfoResp), ErrNotSupported
	}
	return s.Internal.GetPeerInfo(p0)
}

func (s *GatewayApiStub) GetPeerInfo(p0 context.Context) (apitypes.GetPeerInfoResp, error) {
	return *new(apitypes.GetPeerInfoResp), ErrNotSupported
}

func (s *GatewayApiStruct) Load(p0 context.Context, p1 *types.MetadataProposal) (apitypes.LoadResp, error) {
	if s.Internal.Load == nil {
		return *new(apitypes.LoadResp), ErrNotSupported
	}
	return s.Internal.Load(p0, p1)
}

func (s *GatewayApiStub) Load(p0 context.Context, p1 *types.MetadataProposal) (apitypes.LoadResp, error) {
	return *new(apitypes.LoadResp), ErrNotSupported
}

func (s *GatewayApiStruct) NetPeers(p0 context.Context) ([]types.PeerInfo, error) {
	if s.Internal.NetPeers == nil {
		return *new([]types.PeerInfo), ErrNotSupported
	}
	return s.Internal.NetPeers(p0)
}

func (s *GatewayApiStub) NetPeers(p0 context.Context) ([]types.PeerInfo, error) {
	return *new([]types.PeerInfo), ErrNotSupported
}

func (s *GatewayApiStruct) NodeAddress(p0 context.Context) (string, error) {
	if s.Internal.NodeAddress == nil {
		return "", ErrNotSupported
	}
	return s.Internal.NodeAddress(p0)
}

func (s *GatewayApiStub) NodeAddress(p0 context.Context) (string, error) {
	return "", ErrNotSupported
}

func (s *GatewayApiStruct) ShowCommits(p0 context.Context, p1 *types.MetadataProposal) (apitypes.ShowCommitsResp, error) {
	if s.Internal.ShowCommits == nil {
		return *new(apitypes.ShowCommitsResp), ErrNotSupported
	}
	return s.Internal.ShowCommits(p0, p1)
}

func (s *GatewayApiStub) ShowCommits(p0 context.Context, p1 *types.MetadataProposal) (apitypes.ShowCommitsResp, error) {
	return *new(apitypes.ShowCommitsResp), ErrNotSupported
}

func (s *GatewayApiStruct) Test(p0 context.Context, p1 string) (string, error) {
	if s.Internal.Test == nil {
		return "", ErrNotSupported
	}
	return s.Internal.Test(p0, p1)
}

func (s *GatewayApiStub) Test(p0 context.Context, p1 string) (string, error) {
	return "", ErrNotSupported
}

func (s *GatewayApiStruct) Update(p0 context.Context, p1 *types.MetadataProposal, p2 *types.OrderStoreProposal, p3 uint64, p4 []byte) (apitypes.UpdateResp, error) {
	if s.Internal.Update == nil {
		return *new(apitypes.UpdateResp), ErrNotSupported
	}
	return s.Internal.Update(p0, p1, p2, p3, p4)
}

func (s *GatewayApiStub) Update(p0 context.Context, p1 *types.MetadataProposal, p2 *types.OrderStoreProposal, p3 uint64, p4 []byte) (apitypes.UpdateResp, error) {
	return *new(apitypes.UpdateResp), ErrNotSupported
}

var _ GatewayApi = new(GatewayApiStruct)
