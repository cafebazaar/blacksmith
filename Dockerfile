FROM golang:1.5

WORKDIR /go/src/app

ENTRYPOINT ["/usr/local/bin/go-wrapper", "run"]

# Temporary, for faster builds
RUN \
  go get golang.org/x/net/ipv4 && \
  go get github.com/danderson/pixiecore/tftp && \
  go get golang.org/x/net/context && \
  go get github.com/coreos/etcd/client

COPY . /go/src/app
RUN go-wrapper download
RUN go-wrapper install
