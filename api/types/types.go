package apitypes

type CreateResp struct {
	DataId string
	Alias  string
	TxId   string
	Cids   []string
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
