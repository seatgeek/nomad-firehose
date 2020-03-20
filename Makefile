# build config
BUILD_DIR 		?= $(abspath build)
GET_GOARCH 		 = $(word 2,$(subst -, ,$1))
GET_GOOS   		 = $(word 1,$(subst -, ,$1))
GOBUILD   		?= $(shell go env GOOS)-$(shell go env GOARCH)
GOFILES_NOVENDOR = $(shell find . -type f -name '*.go' -not -path "./vendor/*")
VETARGS? 		 =-all
GIT_COMMIT 		:= $(shell git describe --tags)
GIT_DIRTY 		:= $(if $(shell git status --porcelain),+CHANGES)
GO_LDFLAGS 		:= "-X main.GitCommit=$(GIT_COMMIT)$(GIT_DIRTY)"

$(BUILD_DIR):
	mkdir -p $@

# Install all go dependencies via "go get". Will use the "go.mod / go.sum" files automatically
.PHONY: dependencies
dependencies:
	@echo "==> go mod download"
	@go mod download

# Create pseudo Make target for all cmd/ (or GOBUILD) provided
# allow for "make cache_primer" to work automatically
BINARIES = $(addprefix $(BUILD_DIR)/nomad-firehose-, $(GOBUILD))
$(BINARIES): $(BUILD_DIR)/nomad-firehose-%: $(BUILD_DIR) dependencies
	@echo "==> building $@ ..."
	GOOS=$(call GET_GOOS,$*) GOARCH=$(call GET_GOARCH,$*) CGO_ENABLED=0 go build -o $@ -ldflags $(GO_LDFLAGS)

# Build all binaries (or APP_SERVER_NAME) and write to the BIN_DIR/
.PHONY: build
build:
	@$(MAKE) -j $(BINARIES)

# Format go source code
.PHONY: fmt
fmt:
	gofmt -w .

# vet go source code
.PHONY: vet
vet:
	go vet ./...

# Run full test suite
.PHONY: test
test: dependencies
	@echo "==> go test"
	@go test -v -covermode=count ./...

# Build local Dockerfile
.PHONY: docker-build
docker-build:
	@echo "==> Docker build"
	docker build -t nomad-firehose-local .

# Start docker shell
.PHONY: docker-shell
docker-shell: docker-build
	@echo "==> Docker run"
	@docker run --rm -it nomad-firehose-local bash

.PHONY: docker-release
docker-release: docker-build
	@echo "=> build and push Docker image ..."
	docker tag nomad-firehose-local seatgeek/nomad-firehose:$(TAG)
	docker push seatgeek/nomad-firehose:$(TAG)
