package gql

import (
	"context"
	"fmt"
	"sao-node/node/indexer/gql/types"

	"github.com/google/uuid"
	"github.com/graph-gophers/graphql-go"
)

type NFTMetaInfo struct {
	Id        types.Uint64
	NftId     string
	TraitType string
	Value     string
}

type NFTMetaInfoList struct {
	TotalCount   int32
	NFTMetaInfos []*NFTMetaInfo
	More         bool
}

// query: metadata(id) Metadata
func (r *resolver) NFTMetaInfo(ctx context.Context, args struct{ ID graphql.ID }) (*NFTMetaInfo, error) {
	var nftId uuid.UUID
	err := nftId.UnmarshalText([]byte(args.ID))
	if err != nil {
		return nil, fmt.Errorf("parsing graphql ID '%s' as UUID: %w", args.ID, err)
	}

	row := r.indexSvc.Db.QueryRowContext(ctx, "SELECT ID, NFTID, TRAIT_TYPE, VALUE FROM METADATA WHERE NFTID="+nftId.String())
	var Id types.Uint64
	var NftId string
	var TraitType string
	var Value string
	err = row.Scan(&Id, &NftId, &TraitType, &Value)
	if err != nil {
		return nil, err
	}

	return &NFTMetaInfo{
		Id, NftId, TraitType, Value,
	}, nil
}

// query: metadatas(cursor, offset, limit) MetaList
func (r *resolver) NFTMetaInfos(ctx context.Context, args struct{ Query graphql.NullString }) (*NFTMetaInfoList, error) {
	queryStr := "SELECT COMMITID, DID, CID, DATAID, ALIAS, PLAT, VER, SIZE, EXPIRATION, READER, WRITER FROM METADATA "
	if args.Query.Set && args.Query.Value != nil {
		queryStr = queryStr + *args.Query.Value
	}
	rows, err := r.indexSvc.Db.QueryContext(ctx, queryStr)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	nftMetaInfos := make([]*NFTMetaInfo, 0)
	for rows.Next() {
		var Id types.Uint64
		var NftId string
		var TraitType string
		var Value string
		err = rows.Scan(&Id, &NftId, &TraitType, &Value)
		if err != nil {
			return nil, err
		}
		nftMetaInfos = append(nftMetaInfos, &NFTMetaInfo{
			Id, NftId, TraitType, Value,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return &NFTMetaInfoList{
		TotalCount:   int32(len(nftMetaInfos)),
		NFTMetaInfos: nftMetaInfos,
		More:         false,
	}, nil
}

func (r *resolver) NFTMetaInfoCount(ctx context.Context) (int32, error) {
	return 0, nil
}

func (m *NFTMetaInfo) ID() graphql.ID {
	return graphql.ID(m.Id)
}
