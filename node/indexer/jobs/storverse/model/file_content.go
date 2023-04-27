package storverse

type FileContent struct {
	ID          string `json:"id"`
	CreatedAt   uint64  `json:"createdAt"`
	ContentPath string `json:"contentPath"`
	CommitID    string
	DataID      string
	Alias       string
}
