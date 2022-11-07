package client

import (
	"github.com/mattbaird/jsonpatch"
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
