.PHONY: help clean blacksmith docker push test prepare_test prepare_test_ws prepare_test_etcd
help:
	@echo "Please use \`make <target>' where <target> is one of"
	@echo "  dependencies to install the dependencies"
	@echo "  blacksmith   to build the main binary (for linux/amd64)"
	@echo "  docker       to build the docker image"
	@echo "  push         to push the built docker to docker hub"
	@echo "  test         to run unittests"
	@echo "  prepare_test to prepare a workspace and an etcd instance for testing"
	@echo "  clean        to remove generated files"

################################################################
#  Variables

GIT ?= git
GO ?= go
OS ?= linux
ARCH ?= amd64
VERSION ?= $(shell git describe --tags)
COMMIT := $(shell git rev-parse HEAD)
BRANCH := $(shell git rev-parse --abbrev-ref HEAD)
ifeq ($(BRANCH), "master")
        DOCKER_TAG := "latest"
else
        DOCKER_TAG := $(BRANCH)
endif
PRIKEY ?= ~/.ssh/id_rsa
PUBKEY ?= ~/.ssh/id_rsa.pub
BUILD_TIME := $(shell LANG=en_US date +"%F_%T_%z")
DEV_MODE := false
DOCKER_IMAGE ?= quay.io/cafebazaar/blacksmith
ETCD_ENDPOINT ?= http://127.0.0.1:20379

#  Variables (only used for test)
DUMMY_WORKSPACE ?= /tmp/blacksmith/workspaces/test-workspace
ETCD_RELEASE_VERSION ?= v2.3.7

################################################################
#  Tasks

prepare_test_ws:
	rm -rf $(DUMMY_WORKSPACE)
	mkdir -p $(DUMMY_WORKSPACE)
	cp $(PUBKEY) $(DUMMY_WORKSPACE)
	cp $(PRIKEY) $(DUMMY_WORKSPACE)
	echo "cluster-variables:" > $(DUMMY_WORKSPACE)/initial.yaml
	echo "  coreos-version: \"1068.2.0\"" >> $(DUMMY_WORKSPACE)/initial.yaml
	echo "  net-conf: '{\"netmask\": \"255.255.255.0\"}'" >> $(DUMMY_WORKSPACE)/initial.yaml
	echo "ssh-keys:" >> $(DUMMY_WORKSPACE)/initial.yaml
	echo "  current-user: \"$(shell cat $(PUBKEY))\"" >> $(DUMMY_WORKSPACE)/initial.yaml

prepare_test_etcd:
	docker kill blacksmith-test-etcd || echo "wasn't running"
	docker rm blacksmith-test-etcd || echo "didn't exist'"
	docker pull quay.io/coreos/etcd:$(ETCD_RELEASE_VERSION)
	docker run -d -p 127.0.0.1:20380:2380 -p 127.0.0.1:20379:2379 \
	 --name blacksmith-test-etcd quay.io/coreos/etcd:$(ETCD_RELEASE_VERSION) \
	 -name etcd0 \
	 -advertise-client-urls http://127.0.0.1:20379 \
	 -listen-client-urls http://0.0.0.0:2379 \
	 -initial-advertise-peer-urls http://127.0.0.1:20380 \
	 -listen-peer-urls http://0.0.0.0:2380 \
	 -initial-cluster-token etcd-cluster-1 \
 	 -initial-cluster etcd0=http://127.0.0.1:20380 \
	 -initial-cluster-state new

prepare_test: prepare_test_ws prepare_test_etcd

gotest: *.go */*.go pxe/pxelinux_autogen.go templating/files_autogen.go web/ui_autogen.go 
	$(GO) get -t -v ./...
	ETCD_ENDPOINT=$(ETCD_ENDPOINT) $(GO) test -v ./...

dev: DEV_MODE=true
dev: blacksmith docker

production: blacksmith docker		

dependencies: *.go */*.go pxe/pxelinux_autogen.go templating/files_autogen.go web/ui_autogen.go
	$(GO) get -v
	$(GO) list -f=$(FORMAT) $(TARGET) | xargs $(GO) install

blacksmith: *.go */*.go pxe/pxelinux_autogen.go templating/files_autogen.go web/ui_autogen.go swagger blacksmithctl
	GOOS=$(OS) GOARCH=$(ARCH) $(GO) build -ldflags "-s -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.buildTime=$(BUILD_TIME) -X main.debugMode=$(DEV_MODE)" -o blacksmith

templating/files_autogen.go:  templating/files
	$(GO) get github.com/mjibson/esc
	GOOS=$(OS) GOARCH=$(ARCH) $(GO) generate

pxe/pxelinux_autogen.go: pxe/pxelinux
	$(GO) get github.com/mjibson/esc
	GOOS=$(OS) GOARCH=$(ARCH) $(GO) generate


EXTERNAL_FILES := web/static/bower_components/angular/angular.min.js web/static/bower_components/angular-route/angular-route.min.js web/static/bower_components/angular-resource/angular-resource.min.js web/static/bower_components/angular-xeditable/dist/js/xeditable.min.js web/static/bower_components/jquery/dist/jquery.min.js web/static/bower_components/bootstrap/dist/js/bootstrap.min.js web/static/bower_components/bootstrap/dist/css/bootstrap.css web/static/bower_components/angular-xeditable/dist/css/xeditable.css
web/static/external: $(EXTERNAL_FILES)
	mkdir -p web/static/external
	cp $(EXTERNAL_FILES) web/static/external

EXTERNAL_FILES_FONT := web/static/bower_components/bootstrap/fonts/glyphicons-halflings-regular.woff2
web/static/fonts: $(EXTERNAL_FILES_FONT)
	mkdir -p web/static/fonts
	cp $(EXTERNAL_FILES_FONT) web/static/fonts

web/ui_autogen.go: web/static/* web/static/partials/* web/static/css/*  web/static/img/* web/static/js/* web/static/external web/static/fonts
	$(GO) get github.com/mjibson/esc
	GOOS=$(OS) GOARCH=$(ARCH) $(GO) generate

clean:
	rm -rf blacksmith blacksmithctl swagger pxe/pxelinux_autogen.go templating/files_autogen.go web/ui_autogen.go web/static/external web/static/fonts

docker: blacksmith
	docker build -t $(DOCKER_IMAGE):$(DOCKER_TAG) .

push: docker
	docker push $(DOCKER_IMAGE):$(DOCKER_TAG)

blacksmithctl:
	@GOOS=$(OS) GOARCH=$(ARCH) $(GO) build -o blacksmithctl github.com/cafebazaar/blacksmith/cmd/blacksmithctl

swagger:
	$(GO) get github.com/go-swagger/go-swagger/cmd/swagger
	swagger generate server swagger.yaml --target=swagger --exclude-main
	swagger generate client swagger.yaml --target=swagger
	$(GO) get ./swagger/...
