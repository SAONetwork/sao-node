package gql

import (
	"context"
	"github.com/graph-gophers/graphql-go"
	"strings"
)

type node struct {
	Creator         string
	Peer            string
	Reputation      float64
	Status          int32
	LastAliveHeight int32
	TxAddresses     string
	Role            int32
	Validator       string
	IsGateway       bool
	IsSP            bool
	IsIndexer       bool
	IsAlive         bool
	IPAddress       string
	LastAliveTime   int32
	Name            string
}

type nodeList struct {
	TotalCount int32
	Nodes      []*node
	More       bool
}

type nodeCountInfo struct {
	TotalGateway   int32
	OnlineGateway  int32
	OfflineGateway int32
	TotalSP        int32
	OnlineSP       int32
	OfflineSP      int32
}

// query: nodes(isActive: Boolean, isGateway: Boolean, isSP: Boolean) NodeList
func (r *resolver) Nodes(ctx context.Context, args struct {
	IsActive  *bool
	IsGateway *bool
	IsSP      *bool
}) (*nodeList, error) {
	queryStr := "SELECT Creator, Peer, Reputation, Status, LastAliveHeight, TxAddresses, Role, Validator, IsGateway, IsSP, IsIndexer, IsAlive, IPAddress, LastAliveTime, Name FROM NODE"

	var params []interface{}
	var whereClauses []string

	if args.IsActive != nil {
		whereClauses = append(whereClauses, "IsAlive = ?")
		params = append(params, *args.IsActive)
	}
	if args.IsGateway != nil {
		whereClauses = append(whereClauses, "IsGateway = ?")
		params = append(params, *args.IsGateway)
	}
	if args.IsSP != nil {
		whereClauses = append(whereClauses, "IsSP = ?")
		params = append(params, *args.IsSP)
	}

	if len(whereClauses) > 0 {
		queryStr += " WHERE " + strings.Join(whereClauses, " AND ")
	}

	rows, err := r.indexSvc.Db.QueryContext(ctx, queryStr, params...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	nodes := make([]*node, 0)
	for rows.Next() {
		n := &node{}
		err = rows.Scan(&n.Creator, &n.Peer, &n.Reputation, &n.Status, &n.LastAliveHeight, &n.TxAddresses, &n.Role, &n.Validator, &n.IsGateway, &n.IsSP, &n.IsIndexer, &n.IsAlive, &n.IPAddress, &n.LastAliveTime, &n.Name)
		if err != nil {
			return nil, err
		}
		nodes = append(nodes, n)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return &nodeList{
		TotalCount: int32(len(nodes)),
		Nodes:      nodes,
		More:       false,
	}, nil
}

// query: countNodes(id: ID!) NodeCountInfo
func (r *resolver) CountNodes(ctx context.Context, args struct{ ID graphql.ID }) (*nodeCountInfo, error) {
	countInfo := &nodeCountInfo{}

	// Counting total gateways
	err := r.indexSvc.Db.QueryRowContext(ctx, "SELECT COUNT(*) FROM NODE WHERE IsGateway = ?", 1).Scan(&countInfo.TotalGateway)
	if err != nil {
		return nil, err
	}

	// Counting online gateways
	err = r.indexSvc.Db.QueryRowContext(ctx, "SELECT COUNT(*) FROM NODE WHERE IsGateway = ? AND IsAlive = ?", 1, true).Scan(&countInfo.OnlineGateway)
	if err != nil {
		return nil, err
	}

	// Counting offline gateways
	countInfo.OfflineGateway = countInfo.TotalGateway - countInfo.OnlineGateway

	// Counting total SP
	err = r.indexSvc.Db.QueryRowContext(ctx, "SELECT COUNT(*) FROM NODE WHERE IsSP = ?", 1).Scan(&countInfo.TotalSP)
	if err != nil {
		return nil, err
	}

	// Counting online SP
	err = r.indexSvc.Db.QueryRowContext(ctx, "SELECT COUNT(*) FROM NODE WHERE IsSP = ? AND IsAlive = ?", 1, true).Scan(&countInfo.OnlineSP)
	if err != nil {
		return nil, err
	}

	// Counting offline SP
	countInfo.OfflineSP = countInfo.TotalSP - countInfo.OnlineSP

	return countInfo, nil
}
