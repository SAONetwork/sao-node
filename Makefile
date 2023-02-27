SHELL=/usr/bin/env bash

GOCC?=go
BINS:=

ldflags=-X=sao-node/build.CurrentCommit=+git.$(subst -,.,$(shell git describe --always --match=NeVeRmAtCh --dirty 2>/dev/null || git rev-parse --short HEAD 2>/dev/null))
GOFLAGS+=-ldflags="$(ldflags)"

all: saonode saoclient

saonode:
	$(GOCC) build $(GOFLAGS) -o saonode ./cmd/node
.PHONY: saonode
BINS+=saonode

saoclient:
	$(GOCC) build $(GOFLAGS) -o saoclient ./cmd/client
.PHONY: saoclient
BINS+=saoclient

cbor-gen:
	$(GOCC) run ./gen/cbor/cbor_gen.go
.PHONY: cbor-gen

api-gen:
	$(GOCC) run ./gen/api
	goimports -w api
	goimports -w api
.PHONY: api-gen

docsgen-md-bin:
	$(GOCC) build $(GOFLAGS) -o docgen-md ./gen/apidoc

docsgen-md: docsgen-md-bin
	./docgen-md "api/api_gateway.go" "SaoApi" "api" "./api" > docs/api.md

docsgen-cfg:
	$(GOCC) run ./gen/cfgdoc > ./node/config/doc_gen.go

clean:
	rm -rf $(BINS)
.PHONY: clean