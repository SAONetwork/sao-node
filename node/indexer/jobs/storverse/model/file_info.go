package storverse

import "fmt"

type FileInfo struct {
	ID           string `json:"id"`
	CreatedAt    int64  `json:"createdAt"`
	FileDataID   string `json:"fileDataId"`
	ContentType  string `json:"contentType"`
	Owner        string `json:"owner"`
	Filename     string `json:"filename"`
	FileCategory string `json:"fileCategory"`
	ExtendInfo      string `json:"extendInfo"`
	ThumbnailDataId string `json:"thumbnailDataId"`
	CommitID     string
	DataID       string
	Alias        string
}

type FileInfoInsertionStrategy struct{}

func (f FileInfo) InsertValues() string {
	return fmt.Sprintf("('%s','%s','%s', %d, '%s', '%s', '%s', '%s', '%s', '%s', '%s')",
		f.CommitID, f.DataID, f.Alias, f.CreatedAt, f.FileDataID, f.ContentType, f.Owner, f.Filename, f.FileCategory, f.ExtendInfo, f.ThumbnailDataId)
}

func (s FileInfoInsertionStrategy) Convert(item interface{}) BatchInserter {
	return item.(FileInfo)
}

func (s FileInfoInsertionStrategy) TableName() string {
	return "FILE_INFO"
}