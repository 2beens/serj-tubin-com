# Variables
BINARY_OUTPUT = bin
REDIS_PASS ?= <skip>
SERJ_ROOT_PATH = /var/tmp/serj-tubin-com/test_root

# Targets for information and help
.PHONY: info
info:
	@echo "This is supposed to be a small set of tools for my backend service."
	@echo "Usage: make [TARGET]\n"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

.PHONY: help
help: ## Show available commands
	@echo "Usage: make [TARGET]\n"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

# Database setup
.PHONY: db-setup
db-setup: ## Set up the database
	sudo ./scripts/db_setup.sh

# Code generation and building
.PHONY: generate
generate: ## Update generated files
	go generate ./...

# Cleaning
.PHONY: clean
clean: ## Remove the binary output directory
	rm -rf $(BINARY_OUTPUT)

.PHONY: build
build: ## Build the main service
	go build -o $(BINARY_OUTPUT)/service cmd/service/main.go

.PHONY: build-fs
build-fs: ## Build the file service
	go build -o $(BINARY_OUTPUT)/file-box-service cmd/file_service/main.go

.PHONY: build-netlog-backup
build-netlog-backup: ## Build the netlog backup
	go build -o $(BINARY_OUTPUT)/netlog-backup cmd/netlog_gd_backup/main.go

.PHONY: build-all
build-all: build build-fs build-netlog-backup ## Build all services

# Run services
.PHONY: run-service
run-service: ## Run the main service
	go run cmd/service/main.go

.PHONY: run-fs-service
run-fs-service: ## Run the file service
	export SERJ_REDIS_PASS=$(REDIS_PASS) && go run cmd/file_service/main.go -rootpath $(SERJ_ROOT_PATH) -log-file-path=''

.PHONY: run-netlog-backup
run: run-service ## Alias to run the main service

.PHONY: run-fsc
run-fs: run-fs-service ## Alias to run the file service

# Testing
.PHONY: test
test: ## Run unit tests
	go test -v -race ./...

.PHONY: test-all
test-all: ## Run all unit tests with tags
	go test -v -race ./... -tags=all_tests

.PHONY: integration-tests
integration-tests: ## Run integration tests
	ST_INT_TESTS=1 go test -v -race github.com/2beens/serjtubincom/test

# Deployment
.PHONY: deploy
deploy: ## Deploy the main service
	./scripts/redeploy.sh

.PHONY: deploy-c
deploy-c: ## Deploy the main service with current commit
	./scripts/redeploy.sh --current-commit

.PHONY: deploy-file-service
deploy-file-service: ## Deploy the file service
	./scripts/redeploy-file-box-service.sh

.PHONY: deploy-file-service-c
deploy-file-service-c: ## Deploy the file service with current commit
	./scripts/redeploy-file-box-service.sh --current-commit

.PHONY: deploy-fsc
deploy-fsc: deploy-file-service-c ## Alias for deploying file service with current commit

.PHONY: deploy-fs
deploy-fs: deploy-file-service ## Alias for deploying file service

# Compilation for various platforms
.PHONY: compile
compile: ## Compile for various OS and platforms
	echo "Compiling for various OS and Platform"
	go build -o $(BINARY_OUTPUT)/service cmd/service/main.go
	GOOS=linux GOARCH=arm go build -o $(BINARY_OUTPUT)/linux-arm/main main.go
	GOOS=linux GOARCH=arm64 go build -o $(BINARY_OUTPUT)/linux-arm64/main main.go
	GOOS=freebsd GOARCH=386 go build -o $(BINARY_OUTPUT)/freebsd-386/main main.go
	# ... Add other platforms as needed
