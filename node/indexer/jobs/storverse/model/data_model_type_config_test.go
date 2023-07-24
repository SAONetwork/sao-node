package storverse_test

import (
	storverse "github.com/SaoNetwork/sao-node/node/indexer/jobs/storverse/model"
	"testing"

)

func TestAliasInTypeConfigs(t *testing.T) {
	testCases := []struct {
		metaAlias    string
		expectedMatch bool
	}{
		{"verse_comment_like_xxx", true},
		{"verse_comment_xxx", true},
		{"verse_comment_like_comment_xxx", true},
		{"verse_xxxx", true},
		{"purchase_order_xxxx", false},
	}

	for _, testCase := range testCases {
		match := storverse.AliasInTypeConfigs(testCase.metaAlias, storverse.TypeConfigs)
		if match != testCase.expectedMatch {
			t.Errorf("AliasInTypeConfigs(%q) = %v, want %v", testCase.metaAlias, match, testCase.expectedMatch)
		}
	}
}

func TestGetTableNameForAlias(t *testing.T) {
	testCases := []struct {
		metaAlias      string
		expectedTableName string
		expectedMatch   bool
	}{
		{"verse_comment_like_xxx", "VERSE_COMMENT_LIKE", true},
		{"verse_comment_xxx", "VERSE_COMMENT", true},
		{"verse_xxx", "VERSE", true},
		{"verse_like_xxx", "VERSE_LIKE", true},
		{"read_notifications_5xxxxxx", "READ_NOTIFICATIONS", true},
	}

	for _, testCase := range testCases {
		tableName, match := storverse.GetTableNameForAlias(testCase.metaAlias, storverse.TypeConfigs)
		if tableName != testCase.expectedTableName || match != testCase.expectedMatch {
			t.Errorf("GetTableNameForAlias(%q) = (%q, %v), want (%q, %v)", testCase.metaAlias, tableName, match, testCase.expectedTableName, testCase.expectedMatch)
		}
	}
}
