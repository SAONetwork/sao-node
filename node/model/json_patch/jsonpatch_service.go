package json_patch

import (
	"sync"

	applier "github.com/evanphx/json-patch"
	creator "github.com/mattbaird/jsonpatch"
	"golang.org/x/xerrors"
)

type JsonpatchSvc struct {
}

var (
	jsonpatchSvc *JsonpatchSvc
	once         sync.Once
)

func NewJsonpatchSvc() *JsonpatchSvc {
	once.Do(func() {
		jsonpatchSvc = &JsonpatchSvc{}
	})
	return jsonpatchSvc
}

func (svc *JsonpatchSvc) CreatePatch(jsonDataOrg []byte, jsonDataNew []byte) ([]byte, error) {
	patchs, err := creator.CreatePatch(jsonDataOrg, jsonDataNew)
	if err != nil {
		return nil, xerrors.Errorf(err.Error())
	}

	operations := "["
	for _, operation := range patchs {
		if operations != "[" {
			operations += ","
		}
		operations += operation.Json()
	}
	operations += "]"

	return []byte(operations), nil
}

func (svc *JsonpatchSvc) ApplyPatch(jsonDataOrg []byte, patch []byte) ([]byte, error) {
	patcher, err := applier.DecodePatch(patch)
	if err != nil {
		return nil, xerrors.Errorf(err.Error())
	}

	target, err := patcher.Apply(jsonDataOrg)
	if err != nil {
		return nil, xerrors.Errorf(err.Error())
	}

	return target, nil
}
