package storverse

import "fmt"

type FileInfo struct {
	ID          string `json:"id"`
	CreatedAt   int64  `json:"createdAt"`
	FileDataID  string `json:"fileDataId"`
	ContentType string `json:"contentType"`
	Owner       string `json:"owner"`
	Filename    string `json:"filename"`
	FileCategory string `json:"fileCategory"`
	CommitID string
	DataID string
}

func (f FileInfo) InsertValues() string {
	return fmt.Sprintf("('%s','%s', %d, '%s', '%s', '%s', '%s', '%s')",
		f.CommitID, f.DataID, f.CreatedAt, f.FileDataID, f.ContentType, f.Owner, f.Filename, f.FileCategory)
}