package apitypes

type CreateResp struct {
	OrderId uint64
	DataId  string
	Alias   string
	TxId    string
	Cid     string
}

type LoadResp struct {
	OrderId uint64
	DataId  string
	Alias   string
	Content string
}

type DeleteResp struct {
	DataId string
	Alias  string
}
