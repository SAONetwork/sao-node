package apitypes

type CreateResp struct {
	DataId string
	Alias  string
	TxId   string
	Cid    string
}

type UpdateResp struct {
	DataId string
	Alias  string
	TxId   string
	Cid    string
}

type LoadResp struct {
	DataId  string
	Alias   string
	Content string
}

type DeleteResp struct {
	DataId string
	Alias  string
}

type GetPeerInfoResp struct {
	PeerInfo string
}
