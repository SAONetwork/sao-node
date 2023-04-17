package apiclient

import (
	"context"
	"github.com/filecoin-project/go-jsonrpc"
	"net/http"
	"sao-node/api"
)

const (
	namespace = "Sao"
)

func NewGatewayApi(ctx context.Context, address string, token string) (api.SaoApi, jsonrpc.ClientCloser, error) {
	var res api.SaoApiStruct

	//fmt.Println("Sleeping for 8 seconds...")
	//time.Sleep(8 * time.Second) // sleep for 8 seconds
	//fmt.Println("Done.")

	headers := http.Header{}
	headers.Add("Authorization", "Bearer "+string(token))

	closer, err := jsonrpc.NewMergeClient(ctx, address, namespace, api.GetInternalStructs(&res), headers)
	return &res, closer, err
}
