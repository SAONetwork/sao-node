package apitypes

type CreateResp struct {
	OrderId uint64
	DataId  string
	TxId    string
	Cid     string
}

type LoadResp struct {
	OrderId uint64
	DataId  string
	Alias   string
	Content []byte
}
