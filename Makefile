VERSION := $(shell git describe --tags)
COMMIT := $(shell git rev-parse HEAD)
BUILD_TIME := $(shell LANG=en_US date +"%F_%T_%z")
DOCKER_IMAGE ?= "cafebazaar/blacksmith"

test: *.go */*.go pxe/pxelinux_autogen.go web/ui_autogen.go
	go get -t -v ./...
	go test -v ./...

blacksmith: *.go */*.go pxe/pxelinux_autogen.go web/ui_autogen.go
	go get -v
	GOOS=linux GOARCH=amd64 go build -ldflags "-s -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.buildTime=$(BUILD_TIME)" -o blacksmith

pxe/pxelinux_autogen.go: pxe/pxelinux
	go get github.com/jteeuwen/go-bindata/...
	GOOS=linux GOARCH=amd64 go generate

web/ui_autogen.go: web/ui
	go get github.com/jteeuwen/go-bindata/...
	GOOS=linux GOARCH=amd64 go generate

clean:
	rm -f blacksmith pxe/pxelinux_autogen.go web/ui_autogen.go

docker: blacksmith
	docker build -t $(DOCKER_IMAGE) .

push:
	docker push $(DOCKER_IMAGE)

.PHONY: clean docker push test