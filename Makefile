export GO111MODULE := on

init:
	go mod init

download:
	go mod download

vendor:
	go mod vendor

run:
	go run main.go
