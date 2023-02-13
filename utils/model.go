package utils

import (
	"encoding/json"
	"fmt"
	"sao-node/types"

	applier "github.com/evanphx/json-patch"
	creator "github.com/mattbaird/jsonpatch"
)

func GeneratePatch(contentOrigin string, contentTarget string) (string, error) {
	defer func() {
		if err := recover(); err != nil {
			fmt.Println("Invalid input!!!")
		}
	}()

	var model interface{}
	err := json.Unmarshal([]byte(contentOrigin), &model)
	if err != nil {
		return "", types.Wrap(types.ErrUnMarshalFailed, err)
	}

	patchs, err := creator.CreatePatch([]byte(contentOrigin), []byte(contentTarget))
	if err != nil {
		return "", types.Wrap(types.ErrCreatePatchFailed, err)
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
	defer func() {
		if err := recover(); err != nil {
			fmt.Println("Invalid input!!!")
		}
	}()

	patcher, err := applier.DecodePatch(patch)
	if err != nil {
		return nil, types.Wrap(types.ErrCreatePatchFailed, err)
	}

	target, err := patcher.Apply(jsonDataOrg)
	if err != nil {
		return nil, types.Wrap(types.ErrCreatePatchFailed, err)
	}

	return target, nil
}
