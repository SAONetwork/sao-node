package jobs

import (
	"encoding/json"
	"github.com/google/go-cmp/cmp"
	"testing"
)


type UserProfile struct {
	ID              string   `json:"id"`
	CreatedAt       int      `json:"createdAt"`
	UpdatedAt       int      `json:"updatedAt"`
	DID             string   `json:"did"`
	EthAddr         string   `json:"ethAddr"`
	Avatar          string   `json:"avatar"`
	Username        string   `json:"username"`
	FollowingCount  int      `json:"followingCount"`
	Twitter         string   `json:"twitter"`
	Youtube         string   `json:"youtube"`
	Bio             string   `json:"bio"`
	Banner          string   `json:"banner"`
	FollowingDataId []string `json:"followingDataId"`
	CommitID        string   `json:"commitID"`
	DataID          string   `json:"dataID"`
	Alias           string   `json:"alias"`
}

func TestUnmarshal(t *testing.T) {
	tests := []struct {
		name     string
		jsonData []byte
		want     UserProfile
	}{
		{
			name:     "Invalid field (FollowingCount)",
			jsonData: []byte(`{"avatar":"","banner":"","bio":"","createdAt":1683617269788,"did":"did:sid:e11b73cc0c27e13b46a2f567c482171ebc93723dc652922326dd2e6d1e39c435","ethAddr":"","followingCount":"","followingDataId":"","twitter":"","updatedAt":1683617269788,"username":"did:sid:e11b73cc0c27e13b46a2f567c482171ebc93723dc652922326dd2e6d1e39c435","youtube":""}`),
			want: UserProfile{
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
			var got UserProfile
			if err := json.Unmarshal(tt.jsonData, &got); err != nil {
				t.Errorf("Unmarshal() error: %v", err)
			}

			if !cmp.Equal(got, tt.want) {
				t.Errorf("Unmarshal() = %v, want %v", got, tt.want)
			}
		})
	}
}
