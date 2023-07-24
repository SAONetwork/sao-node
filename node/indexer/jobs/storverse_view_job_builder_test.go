package jobs

import (
	"encoding/json"
	"github.com/google/go-cmp/cmp"
	storverse "github.com/SaoNetwork/sao-node/node/indexer/jobs/storverse/model"
	"testing"
)


func TestUnmarshal(t *testing.T) {
	tests := []struct {
		name     string
		jsonData []byte
		want     storverse.UserProfile
	}{
		{
			name:     "Invalid field (FollowingCount)",
			jsonData: []byte(`{"avatar":"","banner":"","bio":"","createdAt":1683617269788,"did":"did:sid:e11b73cc0c27e13b46a2f567c482171ebc93723dc652922326dd2e6d1e39c435","ethAddr":"","followingCount":"","followingDataId":"","twitter":"","updatedAt":1683617269788,"username":"did:sid:e11b73cc0c27e13b46a2f567c482171ebc93723dc652922326dd2e6d1e39c435","youtube":""}`),
			want: storverse.UserProfile{
				Avatar:          "",
				Banner:          "",
				Bio:             "",
				CreatedAt:       1683617269788,
				DID:             "did:sid:e11b73cc0c27e13b46a2f567c482171ebc93723dc652922326dd2e6d1e39c435",
				EthAddr:         "",
				FollowingCount:  0,
				FollowingDataId: []string{""},
				Twitter:         "",
				UpdatedAt:       1683617269788,
				Username:        "did:sid:e11b73cc0c27e13b46a2f567c482171ebc93723dc652922326dd2e6d1e39c435",
				Youtube:         "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got storverse.UserProfile
			if err := json.Unmarshal(tt.jsonData, &got); err != nil {
				t.Errorf("Unmarshal() error: %v", err)
			}

			if !cmp.Equal(got, tt.want) {
				t.Errorf("Unmarshal() = %v, want %v", got, tt.want)
			}
		})
	}
}
