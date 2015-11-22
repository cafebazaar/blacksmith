FROM golang:1.5

WORKDIR /go/src/github.com/cafebazaar/aghajoon

# Temporary, for faster builds
RUN \
  go get -v golang.org/x/net/ipv4 && \
  go get -v golang.org/x/net/context && \
  go get -v gopkg.in/yaml.v2 && \
  go get -v github.com/danderson/pixiecore/tftp && \
  go get -v github.com/coreos/etcd/client && \
  go get -v github.com/krolaw/dhcp4 &&\
  go get -v github.com/gorilla/mux

ENTRYPOINT ["/go/src/github.com/cafebazaar/aghajoon/aghajoon"]
# ENTRYPOINT ["/bin/bash"]

COPY . /go/src/github.com/cafebazaar/aghajoon
RUN go build .
