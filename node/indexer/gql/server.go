package gql

import (
	"context"
	_ "embed"
	"github.com/SaoNetwork/sao-node/chain"
	"net/http"
	"sync"
	"time"

	"github.com/SaoNetwork/sao-node/node/indexer"

	"github.com/graph-gophers/graphql-go"
	"github.com/graph-gophers/graphql-go/relay"
	logging "github.com/ipfs/go-log/v2"
	"github.com/patrickmn/go-cache"
)

var log = logging.Logger("graphql")

type Server struct {
	listenAddr string
	resolver   *resolver
	srv        *http.Server
	wg         sync.WaitGroup
}

func NewGraphqlServer(listenAddr string, indexSvc *indexer.IndexSvc, svc *chain.ChainSvc) *Server {
	c := cache.New(1*time.Minute, 10*time.Minute)
	return &Server{listenAddr: listenAddr, resolver: &resolver{indexSvc: indexSvc, cache: c, chainSvc: svc}}
}

//go:embed schema.graphql
var schemaGraqhql string

func (s *Server) Start(ctx context.Context) error {
	log.Info("graphql server starting...")

	mux := http.NewServeMux()
	mux.HandleFunc("/graphiql", graphiql())

	opts := []graphql.SchemaOpt{graphql.UseFieldResolvers()}
	schema, err := graphql.ParseSchema(string(schemaGraqhql), s.resolver, opts...)
	if err != nil {
		return err
	}

	queryHandler := &relay.Handler{Schema: schema}

	s.srv = &http.Server{Addr: s.listenAddr, Handler: mux}
	log.Infof("graphql server listening on %s", s.listenAddr)
	mux.Handle("/graphql/query", authenticate(&corsHandler{queryHandler}, s.resolver))

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()

		if err := s.srv.ListenAndServe(); err != http.ErrServerClosed {
			log.Fatalf("gql.ListenAndServe(): %v", err)
		}
	}()

	return nil
}

func (s *Server) Stop(ctx context.Context) error {
	if err := s.srv.Shutdown(ctx); err != nil {
		return err
	}

	s.wg.Wait()

	return nil
}