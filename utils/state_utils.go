package utils

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"time"

	"github.com/SaoNetwork/sao-node/types"

	"github.com/ipfs/go-cid"
	"github.com/ipfs/go-datastore"
)

const (
	ORDER_INDEX_KEY        = "order-index"
	ORDER_KEY              = "order-%s"
	SHARD_INDEX_KEY        = "shard-index"
	SHARD_KEY              = "order-%d-shard-%v"
	MIGRATE_INDEX_KEY      = "migrate-index"
	MIGRATE_KEY            = "migrate-dataid-%s-from-%s"
	SHARD_EXPIRE_INDEX_KEY = "shard-expire"
	SHARD_EXPIRE_KEY       = "shard-expire-%d"
	LATEST_SHARD_ID        = "latest-shard-id"
)

// -----
// shard cid
// -----
func shardExpireDatastoreKey(shardId uint64) datastore.Key {
	return datastore.NewKey(fmt.Sprintf(SHARD_EXPIRE_KEY, shardId))
}

func SaveShardExpire(ctx context.Context, ds datastore.Batching, shardId uint64, cid string, orderId uint64) error {
	key := shardExpireDatastoreKey(shardId)

	exists, err := ds.Has(ctx, key)
	if err != nil {
		return err
	}

	expireInfo := types.ShardExpireInfo{
		ShardId: shardId,
		Cid:     cid,
		OrderId: orderId,
	}

	buf := new(bytes.Buffer)
	err = expireInfo.MarshalCBOR(buf)
	if err != nil {
		return err
	}

	err = ds.Put(ctx, key, buf.Bytes())
	if err != nil {
		return err
	}
	if !exists {
		err = AddShardExpireIndex(ctx, ds, shardId)
		if err != nil {
			return err
		}
	}
	return nil
}

func GetShardExpire(ctx context.Context, ds datastore.Batching, shardId uint64) (types.ShardExpireInfo, error) {
	key := shardExpireDatastoreKey(shardId)
	exists, err := ds.Has(ctx, key)
	if err != nil {
		return types.ShardExpireInfo{}, err
	}
	if !exists {
		return types.ShardExpireInfo{}, nil
	}

	bs, err := ds.Get(ctx, key)
	if err != nil {
		return types.ShardExpireInfo{}, err
	}

	var sei types.ShardExpireInfo
	err = sei.UnmarshalCBOR(bytes.NewReader(bs))
	if err != nil {
		return types.ShardExpireInfo{}, err
	}

	return sei, nil
}

func AddShardExpireIndex(ctx context.Context, ds datastore.Batching, id uint64) error {
	key := datastore.NewKey(SHARD_EXPIRE_INDEX_KEY)
	exists, err := ds.Has(ctx, key)
	if err != nil {
		return err
	}
	var index types.ShardExpireIndex
	if exists {
		data, err := ds.Get(ctx, key)
		if err != nil {
			return err
		}
		err = index.UnmarshalCBOR(bytes.NewReader(data))
		if err != nil {
			return err
		}
	}
	index.Alls = append(index.Alls, types.ShardExpireKey{ShardId: id})

	buf := new(bytes.Buffer)
	err = index.MarshalCBOR(buf)
	if err != nil {
		return err
	}
	err = ds.Put(ctx, key, buf.Bytes())
	if err != nil {
		return err
	}
	return nil
}

func RemoveShardExpireIndex(ctx context.Context, ds datastore.Batching, id uint64) error {
	key := datastore.NewKey(SHARD_EXPIRE_INDEX_KEY)
	exists, err := ds.Has(ctx, key)
	if err != nil {
		return err
	}
	var index types.ShardExpireIndex
	if exists {
		data, err := ds.Get(ctx, key)
		if err != nil {
			return err
		}
		err = index.UnmarshalCBOR(bytes.NewReader(data))
		if err != nil {
			return err
		}
	}

	for i, k := range index.Alls {
		if k.ShardId == id {
			index.Alls = append(index.Alls[:i], index.Alls[i+1:]...)
			break
		}
	}

	buf := new(bytes.Buffer)
	err = index.MarshalCBOR(buf)
	if err != nil {
		return err
	}
	err = ds.Put(ctx, key, buf.Bytes())
	if err != nil {
		return err
	}
	return nil
}

func GetShardExpireIndex(ctx context.Context, ds datastore.Batching) (types.ShardExpireIndex, error) {
	key := datastore.NewKey(SHARD_EXPIRE_INDEX_KEY)
	exists, err := ds.Has(ctx, key)
	if err != nil {
		return types.ShardExpireIndex{}, err
	}
	if !exists {
		return types.ShardExpireIndex{}, nil
	}

	data, err := ds.Get(ctx, key)
	if err != nil {
		return types.ShardExpireIndex{}, err
	}

	var index types.ShardExpireIndex
	err = index.UnmarshalCBOR(bytes.NewReader(data))
	return index, err
}

// -----
// order
// -----

/**
 * get order key in datastore.
 */
func orderDatastoreKey(id string) datastore.Key {
	return datastore.NewKey(fmt.Sprintf(ORDER_KEY, id))
}

/**
 * Save order state in datastore.
 */
func SaveOrder(ctx context.Context, ds datastore.Batching, order types.OrderInfo) error {
	key := orderDatastoreKey(order.DataId)

	exists, err := ds.Has(ctx, key)
	if err != nil {
		return err
	}

	buf := new(bytes.Buffer)
	err = order.MarshalCBOR(buf)
	if err != nil {
		return err
	}
	err = ds.Put(ctx, key, buf.Bytes())
	if err != nil {
		return err
	}

	if !exists {
		err = UpdateOrderIndex(ctx, ds, order.DataId)
		if err != nil {
			return err
		}
	}
	return nil
}

/**
 * Get order state from datastore.
 */
func GetOrder(ctx context.Context, ds datastore.Batching, id string) (types.OrderInfo, error) {
	key := orderDatastoreKey(id)
	exists, err := ds.Has(ctx, key)
	if err != nil {
		return types.OrderInfo{}, err
	}
	if !exists {
		return types.OrderInfo{}, nil
	}

	bs, err := ds.Get(ctx, key)
	if err != nil {
		return types.OrderInfo{}, err
	}

	var orderInfo types.OrderInfo
	err = orderInfo.UnmarshalCBOR(bytes.NewReader(bs))
	if err != nil {
		return types.OrderInfo{}, err
	}
	return orderInfo, nil
}

/**
 * update order index.
 */
func UpdateOrderIndex(ctx context.Context, ds datastore.Batching, id string) error {
	key := datastore.NewKey(ORDER_INDEX_KEY)
	exists, err := ds.Has(ctx, key)
	if err != nil {
		return err
	}
	var index types.OrderIndex
	if exists {
		data, err := ds.Get(ctx, key)
		if err != nil {
			return err
		}
		err = index.UnmarshalCBOR(bytes.NewReader(data))
		if err != nil {
			return err
		}
	}
	index.Alls = append(index.Alls, types.OrderKey{DataId: id})

	buf := new(bytes.Buffer)
	err = index.MarshalCBOR(buf)
	if err != nil {
		return err
	}
	err = ds.Put(ctx, key, buf.Bytes())
	if err != nil {
		return err
	}
	return nil
}

/**
 * Get order index.
 */
func GetOrderIndex(ctx context.Context, ds datastore.Batching) (types.OrderIndex, error) {
	key := datastore.NewKey(ORDER_INDEX_KEY)
	exists, err := ds.Has(ctx, key)
	if err != nil {
		return types.OrderIndex{}, err
	}
	if !exists {
		return types.OrderIndex{}, nil
	}

	data, err := ds.Get(ctx, key)
	if err != nil {
		return types.OrderIndex{}, err
	}

	var index types.OrderIndex
	err = index.UnmarshalCBOR(bytes.NewReader(data))
	return index, err
}

// -----
// migrate
// -----
func migrateDatastoreKey(dataId string, from string) datastore.Key {
	return datastore.NewKey(fmt.Sprintf(MIGRATE_KEY, dataId, from))
}

func SaveMigrate(ctx context.Context, ds datastore.Batching, migrate types.MigrateInfo) error {
	key := migrateDatastoreKey(migrate.DataId, migrate.FromProvider)
	exists, err := ds.Has(ctx, key)
	if err != nil {
		return err
	}

	buf := new(bytes.Buffer)
	err = migrate.MarshalCBOR(buf)
	if err != nil {
		return err
	}
	err = ds.Put(ctx, key, buf.Bytes())
	if err != nil {
		return err
	}
	if !exists {
		err = UpdateMigrateIndex(ctx, ds, migrate.DataId, migrate.FromProvider)
		if err != nil {
			return err
		}
	}
	return nil
}

func GetMigrate(ctx context.Context, ds datastore.Batching, dataId string, from string) (types.MigrateInfo, error) {
	key := migrateDatastoreKey(dataId, from)
	exists, err := ds.Has(ctx, key)
	if err != nil {
		return types.MigrateInfo{}, err
	}
	if !exists {
		return types.MigrateInfo{}, nil
	}

	bs, err := ds.Get(ctx, key)
	if err != nil {
		return types.MigrateInfo{}, err
	}

	var migrateInfo types.MigrateInfo
	err = migrateInfo.UnmarshalCBOR(bytes.NewReader(bs))
	if err != nil {
		return types.MigrateInfo{}, err
	}
	return migrateInfo, nil
}

func UpdateMigrateIndex(
	ctx context.Context,
	ds datastore.Batching,
	dataId string,
	from string,
) error {
	key := datastore.NewKey(MIGRATE_INDEX_KEY)
	exists, err := ds.Has(ctx, key)
	if err != nil {
		return err
	}

	var index types.MigrateIndex
	if exists {
		data, err := ds.Get(ctx, key)
		if err != nil {
			return err
		}
		err = index.UnmarshalCBOR(bytes.NewReader(data))
		if err != nil {
			return err
		}
	}
	index.All = append(index.All, types.MigrateKey{
		DataId:       dataId,
		FromProvider: from,
	})

	buf := new(bytes.Buffer)
	err = index.MarshalCBOR(buf)
	if err != nil {
		return err
	}
	err = ds.Put(ctx, key, buf.Bytes())
	if err != nil {
		return err
	}
	return nil
}

func GetMigrateIndex(ctx context.Context, ds datastore.Batching) (types.MigrateIndex, error) {
	key := datastore.NewKey(MIGRATE_INDEX_KEY)
	exists, err := ds.Has(ctx, key)
	if err != nil {
		return types.MigrateIndex{}, err
	}
	if !exists {
		return types.MigrateIndex{}, nil
	}

	data, err := ds.Get(ctx, key)
	if err != nil {
		return types.MigrateIndex{}, err
	}

	var index types.MigrateIndex
	err = index.UnmarshalCBOR(bytes.NewReader(data))
	return index, err
}

// -----
// shard
// -----
/**
 * get shard key in datastore.
 */
func orderShardDatastoreKey(orderId uint64, cid cid.Cid) datastore.Key {
	return datastore.NewKey(fmt.Sprintf(SHARD_KEY, orderId, cid))
}

/**
 * save order shard state.
 */
func SaveShard(ctx context.Context, ds datastore.Batching, shard types.ShardInfo) error {
	key := orderShardDatastoreKey(shard.OrderId, shard.Cid)

	exists, err := ds.Has(ctx, key)
	if err != nil {
		return err
	}

	buf := new(bytes.Buffer)
	err = shard.MarshalCBOR(buf)
	if err != nil {
		return err
	}
	err = ds.Put(ctx, key, buf.Bytes())
	if err != nil {
		return err
	}
	if !exists {
		err = UpdateShardIndex(ctx, ds, shard.OrderId, shard.Cid)
		if err != nil {
			return err
		}
	}
	return nil
}

/**
 * Get shard state from datastore.
 */
func GetShard(ctx context.Context, ds datastore.Batching, orderId uint64, cid cid.Cid) (types.ShardInfo, error) {
	key := orderShardDatastoreKey(orderId, cid)
	exists, err := ds.Has(ctx, key)
	if err != nil {
		return types.ShardInfo{}, err
	}
	if !exists {
		return types.ShardInfo{}, nil
	}

	bs, err := ds.Get(ctx, key)
	if err != nil {
		return types.ShardInfo{}, err
	}

	var shardInfo types.ShardInfo
	err = shardInfo.UnmarshalCBOR(bytes.NewReader(bs))
	if err != nil {
		return types.ShardInfo{}, err
	}
	return shardInfo, nil
}

/**
 * update shard index
 */
func UpdateShardIndex(
	ctx context.Context,
	ds datastore.Batching,
	orderId uint64,
	cid cid.Cid,
) error {
	key := datastore.NewKey(SHARD_INDEX_KEY)
	exists, err := ds.Has(ctx, key)
	if err != nil {
		return err
	}

	var index types.ShardIndex
	if exists {
		data, err := ds.Get(ctx, key)
		if err != nil {
			return err
		}
		err = index.UnmarshalCBOR(bytes.NewReader(data))
		if err != nil {
			return err
		}
	}
	index.All = append(index.All, types.ShardKey{
		OrderId: orderId,
		Cid:     cid,
	})

	buf := new(bytes.Buffer)
	err = index.MarshalCBOR(buf)
	if err != nil {
		return err
	}
	err = ds.Put(ctx, key, buf.Bytes())
	if err != nil {
		return err
	}
	return nil
}

/**
 * Get shard index from data store.
 */
func GetShardIndex(ctx context.Context, ds datastore.Batching) (types.ShardIndex, error) {
	key := datastore.NewKey(SHARD_INDEX_KEY)
	exists, err := ds.Has(ctx, key)
	if err != nil {
		return types.ShardIndex{}, err
	}
	if !exists {
		return types.ShardIndex{}, nil
	}

	data, err := ds.Get(ctx, key)
	if err != nil {
		return types.ShardIndex{}, err
	}

	var index types.ShardIndex
	err = index.UnmarshalCBOR(bytes.NewReader(data))
	return index, err
}

/**
 * Save latest shard id that storage ready to check into data store.
 */
func SaveLatestShardId(ctx context.Context, ds datastore.Batching, shardId uint64) error {
	key := datastore.NewKey(LATEST_SHARD_ID)

	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, shardId)
	err := ds.Put(ctx, key, buf)
	if err != nil {
		return err
	}
	return nil
}

/**
 * Get latest shard id that storage ready to check from data store.
 */
func GetLatestShardId(ctx context.Context, ds datastore.Batching) (uint64, error) {
	key := datastore.NewKey(LATEST_SHARD_ID)
	exists, err := ds.Has(ctx, key)
	if err != nil || !exists {
		return 0, err
	}

	data, err := ds.Get(ctx, key)
	if err != nil {
		return 0, err
	}
	return binary.BigEndian.Uint64(data), nil
}

const RetryIntervalCoeff time.Duration = 3

/**
 * Get order retry timestamp.
 */
func GetRetryAt(tries uint64) int64 {
	retryInterval := time.Second
	for i := uint64(0); i < tries; i++ {
		retryInterval *= RetryIntervalCoeff
	}
	return time.Now().Add(retryInterval).Unix()
}
