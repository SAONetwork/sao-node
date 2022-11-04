package client

import (
	cid "github.com/ipfs/go-cid"
	"github.com/mattbaird/jsonpatch"
	"github.com/multiformats/go-multicodec"
	"github.com/multiformats/go-multihash"
	"golang.org/x/xerrors"
)

func GeneratePatch(contentOrigin string, contentTarget string) (string, error) {
	patchs, err := jsonpatch.CreatePatch([]byte(contentOrigin), []byte(contentTarget))
	if err != nil {
		return "", xerrors.Errorf(err.Error())
	}

	operations := "["
	for _, operation := range patchs {
		if operations != "[" {
			operations += ","
		}
		operations += operation.Json()
	}
	operations += "]"

	return operations, nil
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
