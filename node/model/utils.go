package model

import (
	jsoniter "github.com/json-iterator/go"
	uuid "github.com/satori/go.uuid"
	"golang.org/x/xerrors"
)

func GenerateResourceId() string {
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
