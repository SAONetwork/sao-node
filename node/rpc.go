package node

import (
	"github.com/filecoin-project/go-jsonrpc"
	"github.com/filecoin-project/go-jsonrpc/auth"
	"github.com/gorilla/mux"
	logging "github.com/ipfs/go-log/v2"
	"github.com/multiformats/go-multiaddr"
	manet "github.com/multiformats/go-multiaddr/net"
	"golang.org/x/xerrors"
	"net/http"
	"sao-storage-node/api"
)

var rpclog = logging.Logger("rpc")

func ServeRPC(h http.Handler, addr multiaddr.Multiaddr) (StopFunc, error) {
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

	return srv.Shutdown, err
}

func GatewayRpcHandler(ga api.GatewayApi) (http.Handler, error) {
	m := mux.NewRouter()

	rpcServer := jsonrpc.NewServer()
	rpcServer.Register("Sao", ga)

	m.Handle("/rpc/v0", rpcServer)

	ah := &auth.Handler{
		Next: m.ServeHTTP,
	}
	return ah, nil
}
