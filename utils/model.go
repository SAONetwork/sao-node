package utils

import (
	applier "github.com/evanphx/json-patch"
	creator "github.com/mattbaird/jsonpatch"

	"golang.org/x/xerrors"
)

func GeneratePatch(contentOrigin string, contentTarget string) (string, error) {
	patchs, err := creator.CreatePatch([]byte(contentOrigin), []byte(contentTarget))
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

func ApplyPatch(jsonDataOrg []byte, patch []byte) ([]byte, error) {
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
