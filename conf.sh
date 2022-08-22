#!/usr/bin/env bash
# WARNING Do not set exit on error or undefined or pipefail!
# This script is intended to be sourced from bash profile

# Helper func to toggle env with github.com/mozey/config
conf() {
    # Script must run in same dir as the config cmd
    if ! test -f "${GOPATH}/bin/configu"; then
        echo "${GOPATH}/bin/configu not found"
        return 1
    fi

    # APP_DIR is the full path to the config cmd basedir.
    # Project files can be referenced relative to APP_DIR
    APP_DIR=$(pwd)
    export APP_DIR

    # Default env is dev, first arg overrides
    ENV=""
    if [[ $# -eq 1 ]]; then
        ENV=${1}
    fi
    if [[ -z "${ENV}" ]]; then
        ENV="dev"
    fi

    # Set env as per config file
    if OUTPUT="$("${GOPATH}"/bin/configu -os linux -env ${ENV})"; then
        eval "$OUTPUT"
        eval "export APP_DIR=$(pwd)"
        # Checking retVal with $? won't work here
        printenv | sort | grep --color -E "APP_|AWS_"
        return 0
    else
        echo "$OUTPUT"
        return 1
    fi
}
