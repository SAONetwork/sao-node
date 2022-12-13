package schema_helper

import uuid "github.com/satori/go.uuid"

const (
	SAO_LINK_PREFIX = "sao://"
)

type SchemaHelper struct {
	// CacheSvc     *cache.CacheSvc
	//CommitSvc *commit.CommitSvc
}

func GenerateDataId(modelType string, headcommit string, alias string) string {
	return uuid.FromStringOrNil(modelType + headcommit + alias).String()
}

func GenerateDataLink(DataId string) string {
	return SAO_LINK_PREFIX + DataId
}

func FetchContent(link string) (interface{}, error) {
	return nil, nil
}
