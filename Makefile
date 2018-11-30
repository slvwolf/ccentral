PKGS = $(shell go list ./... | grep -v /vendor/)
GO ?= go
TMP_PATH ?= /tmp/gopath

all: test build

get:
	$(GO) get -u github.com/gorilla/mux
	$(GO) get -u github.com/coreos/etcd/client
	$(GO) get -u github.com/pkg/errors
	$(GO) get -u github.com/stretchr/testify/assert
	
test:
	$(GO) test $(PKGS)

build:
	$(GO) build

static_linux:
	rm -rf ${TMP_PATH}
	mkdir ${TMP_PATH}
	mkdir -p ${TMP_PATH}/src/github.com/slvwolf/ccentral
	GOPATH=${TMP_PATH} go get -d -u -v \
		github.com/gorilla/mux \
		github.com/coreos/etcd/client \
		github.com/pkg/errors
	cp -r * ${TMP_PATH}/src/github.com/slvwolf/ccentral
	GOPATH=${TMP_PATH} env GOOS=linux GOARCH=amd64 $(GO) build -a -ldflags '-s' -tags netgo -installsuffix netgo -v -o ccentral