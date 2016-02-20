VERSION ?= $(shell git describe --tags)
COMMIT := $(shell git rev-parse HEAD)
BUILD_TIME := $(shell LANG=en_US date +"%F_%T_%z")
DOCKER_IMAGE ?= "cafebazaar/blacksmith"

.PHONY: help clean docker push test
help:
	@echo "Please use \`make <target>' where <target> is one of"
	@echo "  blacksmith   to build the main binary (for linux/amd64)"
	@echo "  docker       to build the docker image"
	@echo "  push         to push the built docker to docker hub"
	@echo "  test         to run unittests"
	@echo "  clean        to remove generated files"

test: *.go */*.go pxe/pxelinux_autogen.go web/ui_autogen.go
	go get -t -v ./...
	go test -v ./...

blacksmith: *.go */*.go pxe/pxelinux_autogen.go web/ui_autogen.go
	go get -v
	GOOS=linux GOARCH=amd64 go build -ldflags "-s -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.buildTime=$(BUILD_TIME)" -o blacksmith

pxe/pxelinux_autogen.go: pxe/pxelinux
	go get -v github.com/mjibson/esc
	GOOS=linux GOARCH=amd64 go generate

EXTERNAL_FILES := web/ui/bower_components/angular/angular.min.js web/ui/bower_components/angular-route/angular-route.min.js web/ui/bower_components/angular-resource/angular-resource.min.js web/ui/bower_components/angular-xeditable/dist/js/xeditable.min.js web/ui/bower_components/jquery/dist/jquery.min.js web/ui/bower_components/bootstrap/dist/js/bootstrap.min.js web/ui/bower_components/bootstrap/dist/css/bootstrap.css web/ui/bower_components/angular-xeditable/dist/css/xeditable.css
web/ui/external: $(EXTERNAL_FILES)
	mkdir -p web/ui/external
	cp -v $(EXTERNAL_FILES) web/ui/external

EXTERNAL_FILES_FONT := web/ui/bower_components/bootstrap/fonts/glyphicons-halflings-regular.woff2
web/ui/fonts: $(EXTERNAL_FILES_FONT)
	mkdir -p web/ui/fonts
	cp -v $(EXTERNAL_FILES_FONT) web/ui/fonts

web/ui_autogen.go: web/ui/* web/ui/partials/* web/ui/css/*  web/ui/img/* web/ui/js/* web/ui/external web/ui/fonts
	go get -v github.com/mjibson/esc
	GOOS=linux GOARCH=amd64 go generate

clean:
	rm -rf blacksmith pxe/pxelinux_autogen.go web/ui_autogen.go web/ui/external web/ui/fonts

docker: blacksmith
	docker build -t $(DOCKER_IMAGE) .

push: docker
	docker push $(DOCKER_IMAGE)
