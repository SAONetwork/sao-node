package model

type ModelMeta struct {
	DataId     string
	Alias      string
	Tags       []string
	Content    []byte
	Cids       []string
	Status     string
	ExtendInfo string
}

// Status - "Pending"/"Active"/"Expired"
