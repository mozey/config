#!/usr/bin/env bash
set -eu # exit on error or undefined variable
bash -c 'set -o pipefail' # return code of first cmd to fail in a pipeline

APP_DIR=${APP_DIR}
cd ${APP_DIR}

# Remove ignored files

rm -f ./pkg/example/config.go
rm -f ./config
rm -f ./config.dev.json
rm -f ./config.prod.json

