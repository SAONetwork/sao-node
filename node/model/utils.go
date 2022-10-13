package model

import (
	"regexp"
	"strings"

	jsoniter "github.com/json-iterator/go"
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

func IsResourceId(content string) bool {
	r, _ := regexp.Compile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

	return r.MatchString(content)
}

func GenerateAlias(content string) string {
	return uuid.FromStringOrNil(content).String()
}

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
