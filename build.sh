#!/usr/bin/env bash

if [ $# != 1 ]; then
    echo "Usage: $0 [Code File Name]"
    exit 0
fi

_BIN_NAME=rdstool

rm ./pkg/*
GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-w -s" -o ./pkg/${_BIN_NAME}_linux_amd64
upx ./pkg/${_BIN_NAME}_linux_amd64
GOOS=windows GOARCH=amd64 go build -trimpath -ldflags="-w -s" -o ./pkg/${_BIN_NAME}_windows_amd64.exe
upx ./pkg/${_BIN_NAME}_windows_amd64.exe
GOOS=darwin GOARCH=amd64 go build -trimpath -ldflags="-w -s" -o ./pkg/${_BIN_NAME}_darwin_amd64
upx ./pkg/${_BIN_NAME}_darwin_amd64

# reference
# https://golang.org/cmd/link/
# https://golang.org/cmd/go/#hdr-Compile_packages_and_dependencies
# https://devlights.hatenablog.com/entry/2020/10/21/192615
