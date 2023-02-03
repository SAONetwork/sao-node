package utils

import (
	"bytes"
	"context"
	"fmt"
	"regexp"
	"sao-node/types"
	"strings"

	"github.com/ipfs/go-cid"
	"github.com/ipfs/go-datastore"
	jsoniter "github.com/json-iterator/go"
	"github.com/multiformats/go-multicodec"
	"github.com/multiformats/go-multihash"
	uuid "github.com/satori/go.uuid"
	"golang.org/x/xerrors"
)

func IsContent(content string) bool {
	r, _ := regexp.Compile(`^\\{.*?\\}$`)

	return r.MatchString(content)
}

func IsLink(content string) bool {
	return strings.Contains(content, `^((http(s))|(ipfs)|(sao)?://.*?$`)
}

func IsDataId(content string) bool {
	_, err := uuid.FromString(content)
	return err == nil
}

func GenerateAlias(content []byte) string {
	return uuid.FromBytesOrNil(content).String()
}

func GenerateDataId() string {
	return uuid.NewV4().String()
}

func GenerateCommitId() string {
	return uuid.NewV4().String()
}

func GenerateGroupId() string {
	return uuid.NewV4().String()
}

func UnMarshal(jsonString []byte, path ...interface{}) (interface{}, error) {
	result := jsoniter.Get(jsonString, path)
	return result.GetInterface(), result.LastError()
}

func Marshal(obj interface{}) ([]byte, error) {
	b, err := jsoniter.Marshal(obj)

	if err != nil {
		return nil, xerrors.Errorf(err.Error())
	}

	return b, nil
}

func CalculateCid(content []byte) (cid.Cid, error) {
	pref := cid.Prefix{
		Version:  0,
		Codec:    uint64(multicodec.Raw),
		MhType:   multihash.SHA2_256,
		MhLength: -1, // default length
	}

	contentCid, err := pref.Sum(content)
	if err != nil {
		return cid.Undef, err
	}

	return contentCid, nil
}

func PutOrderStat(ctx context.Context, ds datastore.Batching, orderId uint64) error {
	key := datastore.NewKey("order_stats")
	exists, err := ds.Has(ctx, key)
	if err != nil {
		return err
	}
	var orderStats types.OrderStats
	if exists {
		data, err := ds.Get(ctx, key)
		if err != nil {
			return err
		}
		err = orderStats.UnmarshalCBOR(bytes.NewReader(data))
		if err != nil {
			return err
		}
	}
	orderStats.All = append(orderStats.All, orderId)
	buf := new(bytes.Buffer)
	err = orderStats.MarshalCBOR(buf)
	if err != nil {
		return err
	}
	err = ds.Put(ctx, key, buf.Bytes())
	if err != nil {
		return err
	}
	return nil
}

func PutOrder(ctx context.Context, ds datastore.Batching, order types.OrderInfo) error {
	keyName := fmt.Sprintf("orderinfo-%d", order.OrderId)
	key := datastore.NewKey(keyName)

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
		err = PutOrderStat(ctx, ds, order.OrderId)
		if err != nil {
			return err
		}
	}
	return nil
}

func GetOrder(ctx context.Context, ds datastore.Batching, orderId uint64) (types.OrderInfo, error) {
	keyName := fmt.Sprintf("orderinfo-%d", orderId)
	key := datastore.NewKey(keyName)
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
