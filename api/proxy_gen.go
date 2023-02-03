// Code generated by github.com/SaoNetwork/sao-node/gen/api. DO NOT EDIT.

package api

import (
	"context"
	apitypes "sao-node/api/types"
	"sao-node/types"

	"github.com/filecoin-project/go-jsonrpc/auth"
	"golang.org/x/xerrors"
)

var ErrNotSupported = xerrors.New("method not supported")

type SaoApiStruct struct {
	Internal struct {
		AuthNew func(p0 context.Context, p1 []auth.Permission) ([]byte, error) `perm:"admin"`

		AuthVerify func(p0 context.Context, p1 string) ([]auth.Permission, error) `perm:"none"`

		GenerateToken func(p0 context.Context, p1 string) (apitypes.GenerateTokenResp, error) `perm:"read"`

		GetHttpUrl func(p0 context.Context, p1 string) (apitypes.GetUrlResp, error) `perm:"read"`

		GetIpfsUrl func(p0 context.Context, p1 string) (apitypes.GetUrlResp, error) `perm:"read"`

		GetNetPeers func(p0 context.Context) ([]types.PeerInfo, error) `perm:"read"`

		GetNodeAddress func(p0 context.Context) (string, error) `perm:"read"`

		GetPeerInfo func(p0 context.Context) (apitypes.GetPeerInfoResp, error) `perm:"read"`

		ModelCreate func(p0 context.Context, p1 *types.MetadataProposal, p2 *types.OrderStoreProposal, p3 uint64, p4 []byte) (apitypes.CreateResp, error) `perm:"write"`

		ModelCreateFile func(p0 context.Context, p1 *types.MetadataProposal, p2 *types.OrderStoreProposal, p3 uint64) (apitypes.CreateResp, error) `perm:"write"`

		ModelDelete func(p0 context.Context, p1 *types.OrderTerminateProposal, p2 bool) (apitypes.DeleteResp, error) `perm:"write"`

		ModelLoad func(p0 context.Context, p1 *types.MetadataProposal) (apitypes.LoadResp, error) `perm:"read"`

		ModelRenewOrder func(p0 context.Context, p1 *types.OrderRenewProposal, p2 bool) (apitypes.RenewResp, error) `perm:"write"`

		ModelShowCommits func(p0 context.Context, p1 *types.MetadataProposal) (apitypes.ShowCommitsResp, error) `perm:"read"`

		ModelUpdate func(p0 context.Context, p1 *types.MetadataProposal, p2 *types.OrderStoreProposal, p3 uint64, p4 []byte) (apitypes.UpdateResp, error) `perm:"write"`

		ModelUpdatePermission func(p0 context.Context, p1 *types.PermissionProposal, p2 bool) (apitypes.UpdatePermissionResp, error) `perm:"write"`

		OrderList func(p0 context.Context) ([]types.OrderInfo, error) `perm:"read"`

		OrderStatus func(p0 context.Context, p1 uint64) (types.OrderInfo, error) `perm:"read"`
	}
}

type SaoApiStub struct {
}

func (s *SaoApiStruct) AuthNew(p0 context.Context, p1 []auth.Permission) ([]byte, error) {
	if s.Internal.AuthNew == nil {
		return *new([]byte), ErrNotSupported
	}
	return s.Internal.AuthNew(p0, p1)
}

func (s *SaoApiStub) AuthNew(p0 context.Context, p1 []auth.Permission) ([]byte, error) {
	return *new([]byte), ErrNotSupported
}

func (s *SaoApiStruct) AuthVerify(p0 context.Context, p1 string) ([]auth.Permission, error) {
	if s.Internal.AuthVerify == nil {
		return *new([]auth.Permission), ErrNotSupported
	}
	return s.Internal.AuthVerify(p0, p1)
}

func (s *SaoApiStub) AuthVerify(p0 context.Context, p1 string) ([]auth.Permission, error) {
	return *new([]auth.Permission), ErrNotSupported
}

func (s *SaoApiStruct) GenerateToken(p0 context.Context, p1 string) (apitypes.GenerateTokenResp, error) {
	if s.Internal.GenerateToken == nil {
		return *new(apitypes.GenerateTokenResp), ErrNotSupported
	}
	return s.Internal.GenerateToken(p0, p1)
}

func (s *SaoApiStub) GenerateToken(p0 context.Context, p1 string) (apitypes.GenerateTokenResp, error) {
	return *new(apitypes.GenerateTokenResp), ErrNotSupported
}

func (s *SaoApiStruct) GetHttpUrl(p0 context.Context, p1 string) (apitypes.GetUrlResp, error) {
	if s.Internal.GetHttpUrl == nil {
		return *new(apitypes.GetUrlResp), ErrNotSupported
	}
	return s.Internal.GetHttpUrl(p0, p1)
}

func (s *SaoApiStub) GetHttpUrl(p0 context.Context, p1 string) (apitypes.GetUrlResp, error) {
	return *new(apitypes.GetUrlResp), ErrNotSupported
}

func (s *SaoApiStruct) GetIpfsUrl(p0 context.Context, p1 string) (apitypes.GetUrlResp, error) {
	if s.Internal.GetIpfsUrl == nil {
		return *new(apitypes.GetUrlResp), ErrNotSupported
	}
	return s.Internal.GetIpfsUrl(p0, p1)
}

func (s *SaoApiStub) GetIpfsUrl(p0 context.Context, p1 string) (apitypes.GetUrlResp, error) {
	return *new(apitypes.GetUrlResp), ErrNotSupported
}

func (s *SaoApiStruct) GetNetPeers(p0 context.Context) ([]types.PeerInfo, error) {
	if s.Internal.GetNetPeers == nil {
		return *new([]types.PeerInfo), ErrNotSupported
	}
	return s.Internal.GetNetPeers(p0)
}

func (s *SaoApiStub) GetNetPeers(p0 context.Context) ([]types.PeerInfo, error) {
	return *new([]types.PeerInfo), ErrNotSupported
}

func (s *SaoApiStruct) GetNodeAddress(p0 context.Context) (string, error) {
	if s.Internal.GetNodeAddress == nil {
		return "", ErrNotSupported
	}
	return s.Internal.GetNodeAddress(p0)
}

func (s *SaoApiStub) GetNodeAddress(p0 context.Context) (string, error) {
	return "", ErrNotSupported
}

func (s *SaoApiStruct) GetPeerInfo(p0 context.Context) (apitypes.GetPeerInfoResp, error) {
	if s.Internal.GetPeerInfo == nil {
		return *new(apitypes.GetPeerInfoResp), ErrNotSupported
	}
	return s.Internal.GetPeerInfo(p0)
}

func (s *SaoApiStub) GetPeerInfo(p0 context.Context) (apitypes.GetPeerInfoResp, error) {
	return *new(apitypes.GetPeerInfoResp), ErrNotSupported
}

func (s *SaoApiStruct) ModelCreate(p0 context.Context, p1 *types.MetadataProposal, p2 *types.OrderStoreProposal, p3 uint64, p4 []byte) (apitypes.CreateResp, error) {
	if s.Internal.ModelCreate == nil {
		return *new(apitypes.CreateResp), ErrNotSupported
	}
	return s.Internal.ModelCreate(p0, p1, p2, p3, p4)
}

func (s *SaoApiStub) ModelCreate(p0 context.Context, p1 *types.MetadataProposal, p2 *types.OrderStoreProposal, p3 uint64, p4 []byte) (apitypes.CreateResp, error) {
	return *new(apitypes.CreateResp), ErrNotSupported
}

func (s *SaoApiStruct) ModelCreateFile(p0 context.Context, p1 *types.MetadataProposal, p2 *types.OrderStoreProposal, p3 uint64) (apitypes.CreateResp, error) {
	if s.Internal.ModelCreateFile == nil {
		return *new(apitypes.CreateResp), ErrNotSupported
	}
	return s.Internal.ModelCreateFile(p0, p1, p2, p3)
}

func (s *SaoApiStub) ModelCreateFile(p0 context.Context, p1 *types.MetadataProposal, p2 *types.OrderStoreProposal, p3 uint64) (apitypes.CreateResp, error) {
	return *new(apitypes.CreateResp), ErrNotSupported
}

func (s *SaoApiStruct) ModelDelete(p0 context.Context, p1 *types.OrderTerminateProposal, p2 bool) (apitypes.DeleteResp, error) {
	if s.Internal.ModelDelete == nil {
		return *new(apitypes.DeleteResp), ErrNotSupported
	}
	return s.Internal.ModelDelete(p0, p1, p2)
}

func (s *SaoApiStub) ModelDelete(p0 context.Context, p1 *types.OrderTerminateProposal, p2 bool) (apitypes.DeleteResp, error) {
	return *new(apitypes.DeleteResp), ErrNotSupported
}

func (s *SaoApiStruct) ModelLoad(p0 context.Context, p1 *types.MetadataProposal) (apitypes.LoadResp, error) {
	if s.Internal.ModelLoad == nil {
		return *new(apitypes.LoadResp), ErrNotSupported
	}
	return s.Internal.ModelLoad(p0, p1)
}

func (s *SaoApiStub) ModelLoad(p0 context.Context, p1 *types.MetadataProposal) (apitypes.LoadResp, error) {
	return *new(apitypes.LoadResp), ErrNotSupported
}

func (s *SaoApiStruct) ModelRenewOrder(p0 context.Context, p1 *types.OrderRenewProposal, p2 bool) (apitypes.RenewResp, error) {
	if s.Internal.ModelRenewOrder == nil {
		return *new(apitypes.RenewResp), ErrNotSupported
	}
	return s.Internal.ModelRenewOrder(p0, p1, p2)
}

func (s *SaoApiStub) ModelRenewOrder(p0 context.Context, p1 *types.OrderRenewProposal, p2 bool) (apitypes.RenewResp, error) {
	return *new(apitypes.RenewResp), ErrNotSupported
}

func (s *SaoApiStruct) ModelShowCommits(p0 context.Context, p1 *types.MetadataProposal) (apitypes.ShowCommitsResp, error) {
	if s.Internal.ModelShowCommits == nil {
		return *new(apitypes.ShowCommitsResp), ErrNotSupported
	}
	return s.Internal.ModelShowCommits(p0, p1)
}

func (s *SaoApiStub) ModelShowCommits(p0 context.Context, p1 *types.MetadataProposal) (apitypes.ShowCommitsResp, error) {
	return *new(apitypes.ShowCommitsResp), ErrNotSupported
}

func (s *SaoApiStruct) ModelUpdate(p0 context.Context, p1 *types.MetadataProposal, p2 *types.OrderStoreProposal, p3 uint64, p4 []byte) (apitypes.UpdateResp, error) {
	if s.Internal.ModelUpdate == nil {
		return *new(apitypes.UpdateResp), ErrNotSupported
	}
	return s.Internal.ModelUpdate(p0, p1, p2, p3, p4)
}

func (s *SaoApiStub) ModelUpdate(p0 context.Context, p1 *types.MetadataProposal, p2 *types.OrderStoreProposal, p3 uint64, p4 []byte) (apitypes.UpdateResp, error) {
	return *new(apitypes.UpdateResp), ErrNotSupported
}

func (s *SaoApiStruct) ModelUpdatePermission(p0 context.Context, p1 *types.PermissionProposal, p2 bool) (apitypes.UpdatePermissionResp, error) {
	if s.Internal.ModelUpdatePermission == nil {
		return *new(apitypes.UpdatePermissionResp), ErrNotSupported
	}
	return s.Internal.ModelUpdatePermission(p0, p1, p2)
}

func (s *SaoApiStub) ModelUpdatePermission(p0 context.Context, p1 *types.PermissionProposal, p2 bool) (apitypes.UpdatePermissionResp, error) {
	return *new(apitypes.UpdatePermissionResp), ErrNotSupported
}

func (s *SaoApiStruct) OrderList(p0 context.Context) ([]types.OrderInfo, error) {
	if s.Internal.OrderList == nil {
		return *new([]types.OrderInfo), ErrNotSupported
	}
	return s.Internal.OrderList(p0)
}

func (s *SaoApiStub) OrderList(p0 context.Context) ([]types.OrderInfo, error) {
	return *new([]types.OrderInfo), ErrNotSupported
}

func (s *SaoApiStruct) OrderStatus(p0 context.Context, p1 uint64) (types.OrderInfo, error) {
	if s.Internal.OrderStatus == nil {
		return *new(types.OrderInfo), ErrNotSupported
	}
	return s.Internal.OrderStatus(p0, p1)
}

func (s *SaoApiStub) OrderStatus(p0 context.Context, p1 uint64) (types.OrderInfo, error) {
	return *new(types.OrderInfo), ErrNotSupported
}

var _ SaoApi = new(SaoApiStruct)
