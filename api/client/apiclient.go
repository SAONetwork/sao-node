package apiclient

import (
	"context"
	"fmt"
	"net/http"
	"sao-storage-node/api"

	"github.com/filecoin-project/go-jsonrpc"
)

const (
	namespace = "Sao"
)

func NewGatewayApi(ctx context.Context, address string, token string) (api.GatewayApi, jsonrpc.ClientCloser, error) {
	var res api.GatewayApiStruct

	headers := http.Header{}
	headers.Add("Authorization", "Bearer "+string(token))

	fmt.Printf("address %v\r\n", address)
	fmt.Printf("header %v\r\n", headers)

	closer, err := jsonrpc.NewMergeClient(ctx, address, namespace, api.GetInternalStructs(&res), headers)
	return &res, closer, err
}
