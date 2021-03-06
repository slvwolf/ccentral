FROM scratch

ADD ./ccentral /ccentral
COPY web/ /web/

EXPOSE 3000

ENTRYPOINT ["/ccentral"]

# BUILD BINARY: docker run --rm -v $PWD:/go/src/github.com/Applifier/ccentral -w /go/src/github.com/Applifier/ccentral golang:latest /bin/bash -c "make vendor_get; make static && ! ldd ccentral"
