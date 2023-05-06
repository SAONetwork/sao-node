package storverse

import (
	"reflect"
	"regexp"
	"sort"
)

type DataModelTypeConfig struct {
	TableNameFunc func() string
	RecordType    reflect.Type
}

type InsertionStrategy interface {
	Convert(item interface{}) BatchInserter
	TableName() string
}

type BatchInserter interface {
	InsertValues() string
}

// TypeConfigs is a map of data model type aliases to their table names and record types.
var TypeConfigs = map[string]DataModelTypeConfig{
	"user_profile": {
		TableNameFunc: UserProfileInsertionStrategy{}.TableName,
		RecordType: reflect.TypeOf(UserProfile{}),
	},
	"verse": {
		TableNameFunc: VerseInsertionStrategy{}.TableName,
		RecordType: reflect.TypeOf(Verse{}),
	},
	"fileinfo": {
		TableNameFunc: FileInfoInsertionStrategy{}.TableName,
		RecordType: reflect.TypeOf(FileInfo{}),
	},
	"file_info": {
		TableNameFunc: FileInfoInsertionStrategy{}.TableName,
		RecordType: reflect.TypeOf(FileInfo{}),
	},
	"user_following": {
		TableNameFunc: UserFollowingInsertionStrategy{}.TableName,
		RecordType: reflect.TypeOf(UserFollowing{}),
	},
	"listing_info": {
		TableNameFunc: ListingInfoInsertionStrategy{}.TableName,
		RecordType: reflect.TypeOf(ListingInfo{}),
	},
	"purchase_order": {
		TableNameFunc: PurchaseOrderInsertionStrategy{}.TableName,
		RecordType: reflect.TypeOf(PurchaseOrder{}),
	},
	"verse_comment": {
		TableNameFunc: VerseCommentInsertionStrategy{}.TableName,
		RecordType: reflect.TypeOf(VerseComment{}),
	},
	"verse_comment_like": {
		TableNameFunc: VerseCommentLikeInsertionStrategy{}.TableName,
		RecordType: reflect.TypeOf(VerseCommentLike{}),
	},
}

func AliasInTypeConfigs(metaAlias string, typeConfigs map[string]DataModelTypeConfig) bool {
	for alias := range typeConfigs {
		if match, _ := regexp.MatchString(`^`+alias+`(?:_|-|$)`, metaAlias); match {
			return true
		}
	}
	return false
}

func GetTableNameForAlias(metaAlias string, typeConfigs map[string]DataModelTypeConfig) (string, bool) {
	sortedAliases := sortAliasesByLength(typeConfigs)

	for _, alias := range sortedAliases {
		config := typeConfigs[alias]
		if match, _ := regexp.MatchString(`^`+alias+`(?:_|-|$)`, metaAlias); match {
			return config.TableNameFunc(), true
		}
	}
	if regexp.MustCompile("^filecontent(-|_|$)").MatchString(metaAlias) {
		return "FILE_CONTENT", true
	}
	return "", false
}

func GetMatchingTypeConfig(metaAlias string, typeConfigs map[string]DataModelTypeConfig) (*DataModelTypeConfig, bool) {
	sortedAliases := sortAliasesByLength(typeConfigs)
	for _, alias := range sortedAliases {
		config := typeConfigs[alias]
		if match, _ := regexp.MatchString(`^`+alias+`(?:_|-|$)`, metaAlias); match {
			return &config, true
		}
	}
	return nil, false
}

func sortAliasesByLength(typeConfigs map[string]DataModelTypeConfig) []string {
	aliases := make([]string, 0, len(typeConfigs))
	for alias := range typeConfigs {
		aliases = append(aliases, alias)
	}
	sort.Slice(aliases, func(i, j int) bool {
		return len(aliases[i]) > len(aliases[j])
	})
	return aliases
}

