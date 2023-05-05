#!/usr/bin/env bash
# WARNING Do not set exit on error or undefined or pipefail!
# This script is intended to be sourced from bash profile

# Helper func to toggle env with built in bash functions.
# This script only supports ".env" files, see
# https://github.com/mozey/config#toggling-env
conf() {
    # APP_DIR is the full path to the application basedir.
    # The config file must exist under this path,
    # and project files can be referenced relative to APP_DIR
    APP_DIR=$(pwd)
    export APP_DIR

    # Default env is dev, first arg overrides
    ENV=""
    if [ $# -eq 1 ]; then
        ENV=${1}
    fi
    if [ -z "${ENV}" ]; then
        ENV="dev"
    fi

    # File loading precedence, see
    # https://github.com/mozey/config#file-loading-precedence
    FILE=""
    if [[ ${ENV} == "dev" ]]; then
        # Prefix is optional for ENV == "dev"
        if [ -f "${APP_DIR}/.env" ]; then
            FILE="${APP_DIR}/.env"
        fi
    fi
    if [ -f "${APP_DIR}/${ENV}.env" ]; then
        FILE="${APP_DIR}/${ENV}.env"
    fi

    if [ ! -f "${FILE}" ]; then
        echo "config file not found for env ${ENV}"
        return 1
    fi
    echo "Setting env for ${ENV} from ${FILE}"

    # Set environment variables from file
    # https://stackoverflow.com/a/20909045/639133
    UNAME_STR=$(uname)
    if [ "$UNAME_STR" = 'Linux' ]; then
        export "$(grep -v '^#' "${FILE}" | xargs -d '\n')"
    elif [ "$UNAME_STR" = 'FreeBSD' ] || [ "$UNAME_STR" = 'Darwin' ]; then
        export "$(grep -v '^#' "${FILE}" | xargs -0)"
    else
        echo "unexpected uname ${UNAME_STR}"
        return 1
    fi

    # NOTE Unlike conf.configu.sh, this script does not
    # unset env vars subsequently removed from the config file.
    # To refresh your env start a new terminal session,
    # or use the configu command
    # https://github.com/mozey/config#toggling-env-with-configu

    # Print application env
    printenv | sort | grep --color -E "APP_|AWS_"
    return 0
}
