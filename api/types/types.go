package apitypes

type CreateResp struct {
	OrderId uint64
	DataId  string
	TxId    string
}

type LoadResp struct {
	OrderId uint64
	DataId  string
	Alias   string
	Content []byte
}
