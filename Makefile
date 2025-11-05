.PHONY: build clean  fmt

BINARY=anyproxy

SRC_DIR=cmd/anyproxy
DIST_DIR=./dist

GOOS=$(shell go env GOOS)
GOARCH=$(shell go env GOARCH)

BUILD_ARCH=arm arm64 386 amd64 ppc64le riscv64 \
	mips mips64le mipsle loong64 s390x
BUILD_ARGS=-trimpath -ldflags="-s -w"

build_all: clean $(BUILD_ARCH)
$(BUILD_ARCH):
	@echo "Building Linux $@ ..."
	@mkdir -p $(DIST_DIR)/$@
	@rm -rf $(DIST_DIR)/$@/*
	@CGO_ENABLED=0 GOOS=linux GOARCH=$@ GOMIPS=softfloat go build \
		$(BUILD_ARGS) -o $(DIST_DIR)/$@/$(BINARY) $(SRC_DIR)/*.go
	@chmod +x $(DIST_DIR)/$@/$(BINARY)
	@cp example.yaml $(DIST_DIR)/$@/example.yaml

build: clean
	@echo "Building For " $(GOOS)-$(GOARCH)
	@mkdir -p $(DIST_DIR)
	@rm -rf $(DIST_DIR)/$(BINARY)
	@CGO_ENABLED=0 GOMIPS=softfloat go build \
		$(BUILD_ARGS) -o $(DIST_DIR)/$(BINARY) $(SRC_DIR)/*.go
	@chmod +x $(DIST_DIR)/$(BINARY)
	@cp example.yaml $(DIST_DIR)/example.yaml


fmt:
	@GOOS=linux golangci-lint fmt

clean:
	@go mod tidy
	@rm -rf $DIST_DIR