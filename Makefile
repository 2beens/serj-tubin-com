.PHONY: info
info:
	echo "This is supposed to be a small set of tools for my backend service."

.PHONY: db-setup
db-setup:
	sudo ./scripts/db_setup.sh

.PHONY: clean
clean:
	rm -rf ./bin

.PHONY: build build-fs build-netlog-backup build-all
build:
	go build -o bin/service cmd/service/main.go
build-fs:
	go build -o bin/file-box-service cmd/file_service/main.go
build-netlog-backup:
	go build -o bin/netlog-backup cmd/netlog_gd_backup/main.go
build-all: build build-fs build-netlog-backup

.PHONY: run-service run-file-service run run-fs
run-service:
	go run cmd/service/main.go
run-file-service:
	export SERJ_REDIS_PASS='<skip>' && go run cmd/file_service/main.go -rootpath /Users/serj/Documents/projects/serj-tubin-com/test_root -log-file-path=''
run: run-service
run-fs: run-file-service

.PHONY: test test-all
test:
	go test -v -race ./...
test-all:
	go test -v -race ./... -tags=all_tests

.PHONY: integration-tests
integration-tests:
	ST_INT_TESTS=1 go test -v -race github.com/2beens/serjtubincom/test -tags=integration_tests

.PHONY: deploy deploy-c
deploy:
	./scripts/redeploy.sh
deploy-c:
	./scripts/redeploy.sh --current-commit

.PHONY: deploy-file-service deploy-file-service-c deploy-fsc deploy-fs
deploy-file-service:
	./scripts/redeploy-file-box-service.sh
deploy-file-service-c:
	./scripts/redeploy-file-box-service.sh --current-commit
deploy-fsc: deploy-file-service-c
deploy-fs: deploy-file-service

.PHONY: compile
compile:
	echo "Compiling for various OS and Platform"
	go build -o bin/service cmd/service/main.go
	GOOS=linux GOARCH=arm go build -o bin/main-linux-arm main.go
	GOOS=linux GOARCH=arm64 go build -o bin/main-linux-arm64 main.go
	GOOS=freebsd GOARCH=386 go build -o bin/main-freebsd-386 main.go
	# others ...
