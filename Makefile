# TODO: this is not in the phase I want it to be

info:
	echo "I believe small projects do not really need Makefile, but whattahell."
	echo "This is supposed to be a small set of tools for my backend service."

build:
	go build -o bin/service cmd/service/main.go

run:
	go run cmd/service/main.go

test:
	go test -race ./...

compile:
	echo "Compiling for every OS and Platform"
	GOOS=linux GOARCH=arm go build -o bin/main-linux-arm main.go
	GOOS=linux GOARCH=arm64 go build -o bin/main-linux-arm64 main.go
	GOOS=freebsd GOARCH=386 go build -o bin/main-freebsd-386 main.go
	# others ...
