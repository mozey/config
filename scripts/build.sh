#!/usr/bin/env bash
set -eu # exit on error or undefined variable
bash -c 'set -o pipefail' # return code of first cmd to fail in a pipeline

# Build pre-compiled binaries

APP_DIR=${APP_DIR}

mkdir -p ${APP_DIR}/build/windows/386
GOOS="windows" GOARCH="386" go build -o ${APP_DIR}/build/windows/386/configu.exe ${APP_DIR}/cmd/configu/main.go

mkdir -p ${APP_DIR}/build/linux/386
GOOS="linux" GOARCH="386" go build -o ${APP_DIR}/build/linux/386/configu ${APP_DIR}/cmd/configu/main.go
