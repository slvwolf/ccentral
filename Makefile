GO ?= go
GOPATH := ${PWD}:$(GOPATH)
export GOPATH

all: test build

test:
	$(GO) test

build:
	$(GO) build

static:
	env GOOS=linux GOARCH=amd64 $(GO) build -a -ldflags '-s' -tags netgo -installsuffix netgo -v -o ccentral

vendor_clean:
	rm -dRf ${PWD}/vendor/src

vendor_get: vendor_clean
	GOPATH=${PWD}/vendor go get -d -u -v \
	github.com/gorilla/mux \
	github.com/coreos/etcd/client \
	golang.org/x/net/context
