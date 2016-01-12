VERSION := $(shell git describe --tags)
COMMIT := $(shell git rev-parse HEAD)
BUILD_TIME := $(shell LANG=en_US date +"%F_%T_%z")
DOCKER_IMAGE ?= "cafebazaar/blacksmith"

help:
	@echo "Please use \`make <target>' where <target> is one of"
	@echo "  blacksmith   to build the main binary (for linux/amd64)"
	@echo "  docker       to build the docker image"
	@echo "  push         to push the built docker to docker hub"
	@echo "  test         to run unittests"
	@echo "  clean        to remove generated files"

define run-generate =
	go get github.com/mjibson/esc
	GOOS=linux GOARCH=amd64 go generate
endef

test: *.go */*.go pxe/pxelinux_autogen.go web/ui_autogen.go
	go get -t -v ./...
	go test -v ./...

blacksmith: *.go */*.go pxe/pxelinux_autogen.go web/ui_autogen.go
	go get -v
	GOOS=linux GOARCH=amd64 go build -ldflags "-s -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.buildTime=$(BUILD_TIME)" -o blacksmith

pxe/pxelinux_autogen.go: pxe/pxelinux
	$(run-generate)

web/ui_autogen.go: web/ui
	$(run-generate)

clean:
	rm -f blacksmith pxe/pxelinux_autogen.go web/ui_autogen.go

docker: blacksmith
	docker build -t $(DOCKER_IMAGE) .

push:
	docker push $(DOCKER_IMAGE)

.PHONY: help clean docker push test
