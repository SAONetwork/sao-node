package apiclient

import (
	"context"
	"net/http"
	"sao-storage-node/api"

	"github.com/filecoin-project/go-jsonrpc"
)

const (
	namespace = "Sao"
)

func NewGatewayApi(ctx context.Context, addr string, header http.Header) (api.GatewayApi, jsonrpc.ClientCloser, error) {
	var res api.GatewayApiStruct
	closer, err := jsonrpc.NewMergeClient(ctx, addr, namespace, api.GetInternalStructs(&res), header)
	return &res, closer, err
}
