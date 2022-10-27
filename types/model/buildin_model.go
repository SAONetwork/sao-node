package model

type FileModel struct {
	FileName string
	Tags     []string
	Cid      string
	Content  []byte
}
