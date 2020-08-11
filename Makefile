all:
	mkdir -p bin
	GOPATH=${PWD}/path go build -o bin/SimpleFtpClient cmd/SimpleFtpClient/*
	GOPATH=${PWD}/path GOOS=windows GOARCH=amd64 go build -o bin/SimpleFtpClient.exe cmd/SimpleFtpClient/*

run:
	GOPATH=${PWD}/path go run cmd/SimpleFtpClient/*
