all: check-license build generate test

GITHUB_URL=github.com/brancz/kube-pod-exporter
GOOS?=$(shell uname -s | tr A-Z a-z)
GOARCH?=$(subst x86_64,amd64,$(patsubst i%86,386,$(shell uname -m)))
OUT_DIR=_output
BIN?=kube-pod-exporter
VERSION?=$(shell cat VERSION)
PKGS=$(shell go list ./... | grep -v /vendor/)
DOCKER_REPO=quay.io/brancz/kube-pod-exporter

check-license:
	@echo ">> checking license headers"
	@./scripts/check_license.sh

crossbuild:
	@GOOS=darwin ARCH=amd64 $(MAKE) -s build
	@GOOS=linux ARCH=amd64 $(MAKE) -s build
	@GOOS=windows ARCH=amd64 $(MAKE) -s build

build:
	@$(eval OUTPUT=$(OUT_DIR)/$(GOOS)/$(GOARCH)/$(BIN))
	@echo ">> building for $(GOOS)/$(GOARCH) to $(OUTPUT)"
	@mkdir -p $(OUT_DIR)/$(GOOS)/$(GOARCH)
	@CGO_ENABLED=0 go build --installsuffix cgo -o $(OUTPUT) $(GITHUB_URL)

container: build
	docker build -t $(DOCKER_REPO):$(VERSION) .

test:
	@echo ">> running all tests"
	@go test -i $(PKGS)

generate: embedmd
	@echo ">> generating docs"
	@./scripts/generate-help-txt.sh
	@$(GOPATH)/bin/embedmd -w `find ./ -path ./vendor -prune -o -name "*.md" -print`

embedmd:
	@go get github.com/campoy/embedmd

update-cri:
	@echo ">> updating CRI proto definitions"
	curl https://raw.githubusercontent.com/kubernetes/kubernetes/master/pkg/kubelet/apis/cri/runtime/v1alpha2/api.proto > runtime/api.proto

docker-build:
	@echo ">> building docker image for building"
	@docker build -f scripts/Dockerfile -t $(DOCKER_REPO)-build .

docker-make-proto: docker-build
	docker run --rm -it -v `pwd`:/go/src/$(GITHUB_URL) $(DOCKER_REPO)-build make proto

proto: protoc-gen-gofast
	@echo ">> generating go code from protobuf definitions"
	@cd $(GOPATH)/src;protoc --proto_path=$(GOPATH)/src --gofast_out=plugins=grpc:. $(GITHUB_URL)/runtime/api.proto

protoc-gen-gofast:
	@go get -u -v github.com/gogo/protobuf/protoc-gen-gofast

check-proto: proto
	@echo ">> checking protobuf definitions for changes"
	@git diff --exit-code

.PHONY: all check-license crossbuild build container test generate embedmd
