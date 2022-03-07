info:
	echo "This is supposed to be a small set of tools for my backend service."

db-setup:
	sudo ./scripts/db_setup.sh

build:
	go build -o bin/service cmd/service/main.go

run: run-service
run-service:
	go run cmd/service/main.go

run-fs: run-file-service
run-file-service:
	export SERJ_REDIS_PASS='<skip>' && go run cmd/file_service/main.go -rootpath /Users/serj/Documents/projects/serj-tubin-com/test_root -log-file-path=''

test:
	go test -race ./...

deploy:
	./scripts/redeploy.sh
deploy-c:
	./scripts/redeploy.sh --current-commit

deploy-fsc: deploy-file-service-c
deploy-file-service-c:
	./scripts/redeploy-file-box-service.sh --current-commit
deploy-fs: deploy-file-service
deploy-file-service:
	./scripts/redeploy-file-box-service.sh

compile:
	echo "Compiling for various OS and Platform"
	go build -o bin/service cmd/service/main.go
	GOOS=linux GOARCH=arm go build -o bin/main-linux-arm main.go
	GOOS=linux GOARCH=arm64 go build -o bin/main-linux-arm64 main.go
	GOOS=freebsd GOARCH=386 go build -o bin/main-freebsd-386 main.go
	# others ...
