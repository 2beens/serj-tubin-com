info:
	echo "This is supposed to be a small set of tools for my backend service."

db-setup:
	sudo ./scripts/db_setup.sh

clean:
	rm -rf ./bin

build:
	go build -o bin/service cmd/service/main.go
build-fs:
	go build -o bin/file-box-service cmd/file_service/main.go
build-netlog-backup:
	go build -o bin/netlog-backup cmd/netlog_gd_backup/main.go
build-all: build build-fs build-netlog-backup

run-service:
	go run cmd/service/main.go
run-file-service:
	export SERJ_REDIS_PASS='<skip>' && go run cmd/file_service/main.go -rootpath /Users/serj/Documents/projects/serj-tubin-com/test_root -log-file-path=''
run: run-service
run-fs: run-file-service

test:
	go test -race ./...
test-all:
    go test -race ./... -tags=all_tests

deploy:
	./scripts/redeploy.sh
deploy-c:
	./scripts/redeploy.sh --current-commit

deploy-file-service:
	./scripts/redeploy-file-box-service.sh
deploy-file-service-c:
	./scripts/redeploy-file-box-service.sh --current-commit
deploy-fsc: deploy-file-service-c
deploy-fs: deploy-file-service

compile:
	echo "Compiling for various OS and Platform"
	go build -o bin/service cmd/service/main.go
	GOOS=linux GOARCH=arm go build -o bin/main-linux-arm main.go
	GOOS=linux GOARCH=arm64 go build -o bin/main-linux-arm64 main.go
	GOOS=freebsd GOARCH=386 go build -o bin/main-freebsd-386 main.go
	# others ...
