all:
	mkdir -p bin
	GOPATH=${PWD} go build -o bin/SimpleFtpClient main/*
