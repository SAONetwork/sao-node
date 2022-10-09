SHELL=/usr/bin/env bash

GOCC?=go
BINS:=

all: snode clientcli

snode:
	$(GOCC) build -o snode ./cmd/node
.PHONY: snode
BINS+=snode

clientcli:
	$(GOCC) build -o clientcli ./cmd/clientcli
.PHONY: clientcli
BINS+=clientcli

api-gen:
	$(GOCC) run ./gen/api
	goimports -w api
	goimports -w api
.PHONY: api-gen

clean:
	rm -rf $(BINS)
.PHONY: clean