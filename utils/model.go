package utils

import (
	"encoding/json"

	applier "github.com/evanphx/json-patch"
	creator "github.com/mattbaird/jsonpatch"

	"golang.org/x/xerrors"
)

func GeneratePatch(contentOrigin string, contentTarget string) (string, error) {
	var model interface{}
	err := json.Unmarshal([]byte(contentOrigin), &model)
	if err != nil {
		return "", xerrors.Errorf(err.Error())
	}

	patchs, err := creator.CreatePatch([]byte(contentOrigin), []byte(contentTarget))
	if err != nil {
		return "", xerrors.Errorf(err.Error())
	}

	removeOperations := make([]string, 0)
	otherOperations := make([]string, 0)
	for _, operation := range patchs {
		if operation.Operation == "remove" {
			removeOperations = append(removeOperations, operation.Json())
		} else {
			otherOperations = append(otherOperations, operation.Json())
		}
	}

	operations := "["
	for len(removeOperations) > 0 {
		if operations != "[" {
			operations += ","
		}
		index := len(removeOperations) - 1
		operations += removeOperations[index]
		removeOperations = removeOperations[:index]
	}
	for _, operation := range otherOperations {
		if operations != "[" {
			operations += ","
		}
		operations += operation
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
