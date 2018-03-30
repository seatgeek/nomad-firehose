# build config
BUILD_DIR 		?= build
GET_GOARCH 		 = $(word 2,$(subst -, ,$1))
GET_GOOS   		 = $(word 1,$(subst -, ,$1))
GOBUILD   		?= linux-amd64
GOFILES_NOVENDOR = $(shell find . -type f -name '*.go' -not -path "./vendor/*")
VETARGS? 		 =-all

$(BUILD_DIR):
	mkdir -p $@

.PHONY: install
install:
	go get github.com/kardianos/govendor
	govendor sync

.PHONY: build
build: install
	govendor sync
	go install

.PHONY: fmt
fmt:
	@echo "=> Running go fmt" ;
	@if [ -n "`go fmt ${GOFILES_NOVENDOR}`" ]; then \
		echo "[ERR] go fmt updated formatting. Please commit formatted code first."; \
		exit 1; \
	fi

.PHONY: vet
vet: fmt
	@go tool vet 2>/dev/null ; if [ $$? -eq 3 ]; then \
		go get golang.org/x/tools/cmd/vet; \
	fi

	@echo "=> Running go tool vet $(VETARGS) ${GOFILES_NOVENDOR}"
	@go tool vet $(VETARGS) ${GOFILES_NOVENDOR} ; if [ $$? -eq 1 ]; then \
		echo ""; \
		echo "[LINT] Vet found suspicious constructs. Please check the reported constructs"; \
		echo "and fix them if necessary before submitting the code for review."; \
	fi

BINARIES = $(addprefix $(BUILD_DIR)/nomad-firehose-, $(GOBUILD))
$(BINARIES): $(BUILD_DIR)/nomad-firehose-%: $(BUILD_DIR)
	@echo "=> building $@ ..."
	#GOOS=$(call GET_GOOS,$*) GOARCH=$(call GET_GOARCH,$*) CGO_ENABLED=0 govendor build -o $@
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 govendor build -o $@


.PHONY: dist
dist:
	@echo "=> building ..."
	$(MAKE) -j $(BINARIES)

.PHONY: docker
docker:
	@echo "=> build and push Docker image ..."
	#@docker login -u $(DOCKER_USER) -p $(DOCKER_PASS)
	docker build -f Dockerfile -t seatgeek/nomad-firehose .
	#docker tag seatgeek/nomad-firehose:$(COMMIT) seatgeek/nomad-firehose:$(TAG)
	#docker push seatgeek/nomad-firehose:$(TAG)
