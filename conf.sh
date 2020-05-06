#!/usr/bin/env bash
set -eu # exit on error or undefined variable
bash -c 'set -o pipefail' # return code of first cmd to fail in a pipeline

# Script must run in same dir as the config cmd
if ! test -f "./config"; then
    echo "config cmd not found"
    exit 1
fi

# APP_DIR is the full path to the config cmd basedir.
# Project files can be referenced relative to APP_DIR
export APP_DIR=$(pwd)

# Default env is dev, first arg overrides
ENV=""
if [[ $# -eq 1 ]]; then
    ENV=${1}
fi
if [[ -z "${ENV}" ]]; then
    ENV="dev"
fi

# Set env as per config file
if test -f "./config.${ENV}.json"; then
    eval "$(./config -env ${ENV})"
    # Checking retVal with $? won't work here
    printenv | sort | grep --color -E "APP_|AWS_"
else
    echo "config file not found"
    exit 1
fi

