package apiclient

import (
	"context"
	"net/http"

	"github.com/SaoNetwork/sao-node/api"

	"github.com/filecoin-project/go-jsonrpc"
)

const (
	namespace = "Sao"
)

func NewNodeApi(ctx context.Context, address string, token string) (api.SaoApi, jsonrpc.ClientCloser, error) {
	var res api.SaoApiStruct

	headers := http.Header{}
	headers.Add("Authorization", "Bearer "+string(token))

	closer, err := jsonrpc.NewMergeClient(ctx, address, namespace, api.GetInternalStructs(&res), headers)
	return &res, closer, err
}
