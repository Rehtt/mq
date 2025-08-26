VERSION=$(shell git describe --tags --always)

.PHONY: tidy
tidy:
	go mod tidy

.PHONY: genapi
genapi:
	protoc --go_out=. --go-grpc_out=. api/proto/*.proto

.PHONY: build
build:
	mkdir -p bin/ && go build -ldflags "-s -w -X main.Version=$(VERSION)" -trimpath -o ./bin/ 

.PHONY: build-linux
build-linux:
	GOOS=linux GOARCH=amd64 go build -ldflags="-s -w -X main.Version=$(VERSION)" -trimpath -o bin/mq_linux

