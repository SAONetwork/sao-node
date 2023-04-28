package storverse

import (
	"reflect"
	"regexp"
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
	"fileinfo": {
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
	"listing_info": {
		TableName:  "LISTING_INFO",
		RecordType: reflect.TypeOf(ListingInfo{}),
	},
	"purchase_order": {
		TableName:  "PURCHASE_ORDER",
		RecordType: reflect.TypeOf(PurchaseOrder{}),
	},
}

func AliasInTypeConfigs(metaAlias string, typeConfigs map[string]DataModelTypeConfig) bool {
	for alias := range typeConfigs {
		if regexp.MustCompile("^" + alias + "(-|_|$)").MatchString(metaAlias) {
			return true
		}
	}
	return false
}

func GetTableNameForAlias(metaAlias string, typeConfigs map[string]DataModelTypeConfig) (string, bool) {
	for alias, config := range typeConfigs {
		if regexp.MustCompile("^" + alias + "(-|_|$)").MatchString(metaAlias) {
			return config.TableName, true
		}
	}
	if regexp.MustCompile("^filecontent(-|_|$)").MatchString(metaAlias) {
		return "FILE_CONTENT", true
	}
	return "", false
}