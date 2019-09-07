# Basic go commands
GOOS := $(shell uname -s | awk '{print tolower($$0)}')
GOARCH=amd64
GOENV=env GO111MODULE=on
GOCMD=$(GOENV) go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOTOOL=$(GOCMD) tool
GOTIDY=$(GOCMD) mod tidy

# Binary name
BINARY_NAME=go-proxy

# Project variables.
PROJECT_GIT=github.com/
PROJECT_REPO=deepk777
PROJECT_NAME=go-proxy
PROJECT_PATH=$(PROJECT_GIT)/$(PROJECT_REPO)/$(PROJECT_NAME)

CERTIFICATE_AUTHORITY_DIRECTORY=$(CA_DIR)
SERVER_CERTIFICATE_PATH=$(CERT_PATH)
SERVER_KEY_PATH=$(KEY_PATH)

# DOCKER IMAGE TAG
VERSION=`cat VERSION`
TAG=$(VERSION)


all: fmt test build
.PHONY: all

help:
	@echo
	@echo "Available make targets:"
	@echo
	@echo " build				build go exectuable for linux"
	@echo " run					run go exectuable for darwin"
	@echo " clean				cleans old binary"
	@echo " tidy				add missing and remove unused modules"
	@echo " docker-build		builds docker image"
	@echo " docker-clean		clean docker image"
	@echo

fmt:
	@echo "Running Code Format..."
	@go fmt ./...

.PHONY: tidy
tidy:
	@echo "Adding missing and removing unused modules..."
	@$(GOTIDY)

.PHONY: build
build: tidy
	@echo "Building with version $(VERSION) for OS:$(GOOS)"
	@CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) $(GOBUILD) -a -ldflags="-X '$(PROJECT_PATH)/goproxy.ProxyVersion=$(VERSION)'" -o $(BINARY_NAME) cmd/main.go

.PHONY: clean
clean:
	@echo "Removing binary..."
	@rm -f bin/$(GOOS)/$(BINARY_NAME)

.PHONY: lint
lint:
	@echo "Linting..."
	@go vet -v $$(go list ./... | grep -v /vendor/ )
	@golint ./... || true

.PHONY: run
run: build
	@echo "Executing the binary..."
	@./bin/$(GOOS)/$(BINARY_NAME)

.PHONY: docker-run
docker-run:
	@docker run -p 443:443 -p 5000:5000 -it\
		-v $(CERTIFICATE_AUTHORITY_DIRECTORY):/root/cert.pem\
		-v $(SERVER_CERTIFICATE_PATH):/root/crt.pem\
		-v $(SERVER_KEY_PATH):/root/key.pem\
		$(BINARY_NAME):$(TAG)

.PHONY: docker-build
docker-build:
ifdef TARGET
	@echo "Building docker image"
	@docker build -q -t $(BINARY_NAME):$(VERSION) -f Dockerfile .;
	
else
		@echo "Please provide TARGET arg..."
		exit 1
endif

.PHONY: docker-clean
docker-clean:
	@echo "Cleaning docker image"
	@docker rmi -f $(BINARY_NAME):$(TAG)