.PHONY: help clean docker push test prepare_test_etcd
help:
	@echo "Please use \`make <target>' where <target> is one of"
	@echo "  dependencies to install the dependencies"
	@echo "  blacksmith   to build the main binary (for linux/amd64)"
	@echo "  docker       to build the docker image"
	@echo "  push         to push the built docker to docker hub"
	@echo "  test         to run unittests"
	@echo "  clean        to remove generated files"

################################################################
#  Variables
GIT ?= git
GO ?= go
OS ?= linux
ARCH ?= amd64
COMMIT := $(shell git rev-parse HEAD)
BRANCH := $(shell git rev-parse --abbrev-ref HEAD)
DOCKER_IMAGE ?= localhost:5000/blacksmith:$(BRANCH)
DOCKER_IMAGE_PRODUCTION ?= quay.io/cafebazaar/blacksmith:$(BRANCH)
BUILD_TIME := $(shell LANG=en_US date +"%F_%T_%z")
ETCD_ENDPOINT ?= http://127.0.0.1:20379
DUMMY_WORKSPACE ?= /tmp/blacksmith/workspaces/dummy-workspace
ETCD_RELEASE_VERSION ?= v2.3.7
LD_FLAGS := -s -X main.version=$(BRANCH) -X main.commit=$(COMMIT) -X main.buildTime=$(BUILD_TIME)

################################################################
#  Tasks
/tmp/blacksmith/workspaces/dummy-workspace/initial.yaml:
	@rm -rf $(DUMMY_WORKSPACE)
	@mkdir -p $(DUMMY_WORKSPACE)
	@bash scripts/gen-initial-yaml.bash > $@

prepare_test_etcd:
	@docker kill blacksmith-test-etcd || echo "wasn't running"
	@docker rm blacksmith-test-etcd || echo "didn't exist"
	@docker pull quay.io/coreos/etcd:$(ETCD_RELEASE_VERSION)
	@docker run -d -p 127.0.0.1:20380:2380 -p 127.0.0.1:20379:2379 \
	  --name blacksmith-test-etcd quay.io/coreos/etcd:$(ETCD_RELEASE_VERSION) \
	  -name etcd0 \
	  -advertise-client-urls http://127.0.0.1:20379 \
	  -listen-client-urls http://0.0.0.0:2379 \
	  -initial-advertise-peer-urls http://127.0.0.1:20380 \
	  -listen-peer-urls http://0.0.0.0:2380 \
	  -initial-cluster-token etcd-cluster-1 \
	  -initial-cluster etcd0=http://127.0.0.1:20380 \
	  -initial-cluster-state new

test: dependencies
	ETCD_ENDPOINT=$(ETCD_ENDPOINT) $(GO) test -p=1 $(shell glide novendor)

production: blacksmith docker		

dependencies: *.go */*.go pxe/pxelinux_autogen.go templating/files_autogen.go web/ui_autogen.go /tmp/blacksmith/workspaces/dummy-workspace/initial.yaml swagger
	glide --version 2> /dev/null || curl https://glide.sh/get | sh
	glide -q -y glide.yaml install

blacksmith: dependencies blacksmithctl blacksmith-agent
	GOOS=$(OS) GOARCH=$(ARCH) $(GO) build -ldflags="$(LD_FLAGS)" -o $@

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
	rm -rf blacksmith blacksmithctl blacksmith-agent gofmt.diff swagger pxe/pxelinux_autogen.go templating/files_autogen.go web/ui_autogen.go web/static/external web/static/fonts

docker: blacksmith
	docker build -f Dockerfile -t $(DOCKER_IMAGE) .

docker-production: blacksmith
	docker build -f Dockerfile -t $(DOCKER_IMAGE_PRODUCTION) .

push:
	docker push $(DOCKER_IMAGE)

push-production:
	docker push $(DOCKER_IMAGE_PRODUCTION)

blacksmithctl: swagger
	@GOOS=$(OS) GOARCH=$(ARCH) $(GO) build -o $@ github.com/cafebazaar/blacksmith/cmd/blacksmithctl

blacksmith-agent:
	@GOOS=$(OS) GOARCH=$(ARCH) $(GO) build -o $@ -ldflags="$(LD_FLAGS)" github.com/cafebazaar/blacksmith/cmd/blacksmith-agent

gofmt.diff: *.go */*.go */*/*.go
	@gofmt -d $^ > $@

golint: $(GOLINT_BIN)
	@golint -set_exit_status ./...

gofmt: gofmt.diff
	@if [ -s $< ]; then echo 'gofmt found errors'; false; fi

swagger:
	$(GO) get -v github.com/go-swagger/go-swagger/cmd/swagger
	swagger generate server swagger.yaml --target=swagger --flag-strategy=pflag --exclude-main
	swagger generate client swagger.yaml --target=swagger
	$(GO) get ./swagger/...
