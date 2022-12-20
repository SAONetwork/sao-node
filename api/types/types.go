package apitypes

type LoadReq struct {
	User      string
	KeyWord   string
	PublicKey string
	GroupId   string
	CommitId  string
	Version   string
}

type CreateResp struct {
	DataId string
	Alias  string
	TxId   string
	Cid    string
}

type UpdateResp struct {
	DataId   string
	CommitId string
	Alias    string
	TxId     string
	Cid      string
}

type LoadResp struct {
	DataId   string
	Alias    string
	CommitId string
	Version  string
	Cid      string
	Content  string
}

type DeleteResp struct {
	DataId string
	Alias  string
}

type UpdatePermissionResp struct {
	DataId string
}

type RenewResp struct {
	Results map[string]string
}

type ShowCommitsResp struct {
	DataId  string
	Alias   string
	Commits []string
}

type GetPeerInfoResp struct {
	PeerInfo string
}

type GenerateTokenResp struct {
	Server string
	Token  string
}

type GetUrlResp struct {
	Url string
}
