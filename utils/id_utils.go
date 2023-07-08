package utils

import (
	"regexp"
	"strings"

	"github.com/SaoNetwork/sao-node/types"

	"github.com/ipfs/go-cid"
	jsoniter "github.com/json-iterator/go"
	"github.com/multiformats/go-multicodec"
	"github.com/multiformats/go-multihash"
	uuid "github.com/satori/go.uuid"
)

const NS_URL = "6ba7b811-9dad-11d1-80b4-00c04fd430c8"

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

func GenerateDataId(seed string) string {
	return GenerateCommitId(seed)
}

func GenerateCommitId(seed string) string {
	idv1 := uuid.NewV1().String()
	idv5 := uuid.NewV5(uuid.FromStringOrNil(NS_URL), seed).String()

	return idv1[0:18] + idv5[18:]
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
		return nil, types.Wrap(types.ErrMarshalFailed, err)
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
		return cid.Undef, types.Wrap(types.ErrCalculateCidFailed, err)
	}

	return contentCid, nil
}
