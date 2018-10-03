GO ?= go
GOPATH := ${PWD}/vendor:$(GOPATH)
export GOPATH

all: test build

get:
	$(GO) get github.com/gorilla/mux
	$(GO) get github.com/coreos/etcd/client
	$(GO) get golang.org/x/net/context
	$(GO) get github.com/stretchr/testify/assert
	
test:
	$(GO) test

build:
	$(GO) build

static_linux:
	env GOOS=linux GOARCH=amd64 $(GO) build -a -ldflags '-s' -tags netgo -installsuffix netgo -v -o ccentral
