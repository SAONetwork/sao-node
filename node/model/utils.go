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

func IsDataId(content string) bool {
	log.Infof("content: %s", content)
	_, err := uuid.FromString(content)
	return err == nil
}

func GenerateAlias(content []byte) string {
	return uuid.FromBytesOrNil(content).String()
}

func GenerateDataId() string {
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
