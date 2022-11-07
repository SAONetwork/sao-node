package utils

import (
	"regexp"
	"strings"

	"github.com/ipfs/go-cid"
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

func CaculateCid(content []byte) (cid.Cid, error) {
	pref := cid.Prefix{
		Version:  1,
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
