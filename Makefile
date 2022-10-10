SHELL=/usr/bin/env bash

GOCC?=go
BINS:=

all: saonode saoclient

saonode:
	$(GOCC) build -o saonode ./cmd/node
.PHONY: saonode
BINS+=saonode

saoclient:
	$(GOCC) build -o saoclient ./cmd/client
.PHONY: saoclient
BINS+=saoclient

api-gen:
	$(GOCC) run ./gen/api
	goimports -w api
	goimports -w api
.PHONY: api-gen

clean:
	rm -rf $(BINS)
.PHONY: clean