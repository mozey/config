#!/usr/bin/env bash

# Set (e) exit on error
# Set (u) no-unset to exit on undefined variable
set -eu
# If any command in a pipeline fails,
# that return code will be used as the
# return code of the whole pipeline.
bash -c 'set -o pipefail'

APP_DIR=${APP_DIR}
cd ${APP_DIR}

# Remove ignored files

rm -f ./pkg/example/config.go
rm -f ./config
rm -f ./config.dev.json
rm -f ./config.prod.json

