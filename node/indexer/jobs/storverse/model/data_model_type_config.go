package storverse

import (
	"reflect"
	"strings"
)

type DataModelTypeConfig struct {
	TableName string
	RecordType reflect.Type
}

// TypeConfigs is a map of data model type aliases to their table names and record types.
var TypeConfigs = map[string]DataModelTypeConfig{
	"user_profile": {
		TableName:  "USER_PROFILE",
		RecordType: reflect.TypeOf(UserProfile{}),
	},
	"verse": {
		TableName:  "VERSE",
		RecordType: reflect.TypeOf(Verse{}),
	},
	"fileInfo": {
		TableName:  "FILE_INFO",
		RecordType: reflect.TypeOf(FileInfo{}),
	},
	"file_info": {
		TableName:  "FILE_INFO",
		RecordType: reflect.TypeOf(FileInfo{}),
	},
	"user_following": {
		TableName:  "USER_FOLLOWING",
		RecordType: reflect.TypeOf(UserFollowing{}),
	},
}

func AliasInTypeConfigs(metaAlias string, typeConfigs map[string]DataModelTypeConfig) bool {
	for alias := range typeConfigs {
		if strings.Contains(metaAlias, alias) {
			return true
		}
	}
	return false
}

func GetTableNameForAlias(metaAlias string, typeConfigs map[string]DataModelTypeConfig) (string, bool) {
	for alias, config := range typeConfigs {
		if strings.Contains(metaAlias, alias) {
			return config.TableName, true
		}
	}
	return "", false
}