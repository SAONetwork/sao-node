package node

import (
	"context"
	"net/http"
	"sao-node/api"

	"github.com/filecoin-project/go-jsonrpc"
	"github.com/filecoin-project/go-jsonrpc/auth"
	"github.com/gorilla/mux"
	logging "github.com/ipfs/go-log/v2"
	"github.com/multiformats/go-multiaddr"
	manet "github.com/multiformats/go-multiaddr/net"
	"github.com/rs/cors"
	"golang.org/x/xerrors"
)

var rpclog = logging.Logger("rpc")

func ServeRPC(h http.Handler, addr multiaddr.Multiaddr) (*http.Server, error) {
	// Start listening to the addr; if invalid or occupied, we will fail early.
	lst, err := manet.Listen(addr)
	if err != nil {
		return nil, xerrors.Errorf("could not listen: %w", err)
	}

	// Instantiate the server and start listening.
	srv := &http.Server{
		Handler: h,
	}

	go func() {
		err = srv.Serve(manet.NetListener(lst))
		if err != http.ErrServerClosed {
			rpclog.Warnf("rpc server failed: %s", err)
		}
	}()

	return srv, err
}

func GatewayRpcHandler(ga api.SaoApi, enablePermission bool) (http.Handler, error) {
	m := mux.NewRouter()

	if enablePermission {
		ga = api.PermissionedSaoNodeAPI(ga)
	}

	rpcServer := jsonrpc.NewServer()
	rpcServer.Register("Sao", ga)

	m.Handle("/rpc/v0", rpcServer)

	var handler = &auth.Handler{
		Next: m.ServeHTTP,
	}

	if enablePermission {
		handler.Verify = ga.AuthVerify
	} else {
		handler.Verify = authVerify
	}

	return cors.AllowAll().Handler(handler), nil
}

func authVerify(ctx context.Context, token string) ([]auth.Permission, error) {

	return api.AllPermissions, nil
}
