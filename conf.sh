#!/usr/bin/env bash
# WARNING Do not set exit on error or undefined or pipefail!
# This script is intended to be sourced from bash profile

# Helper func to toggle env with built in bash functions.
# This func only supports ".env" files
conf() {
    # TODO Arg to specify ENV

    # TODO File loading precedence

	# Set environment variables from file
	# https://stackoverflow.com/a/20909045/639133
	UNAME_STR=$(uname)
	if [ "$UNAME_STR" = 'Linux' ]; then
		export "$(grep -v '^#' .env | xargs -d '\n')"
	elif [ "$UNAME_STR" = 'FreeBSD' ] || [ "$UNAME_STR" = 'Darwin' ]; then
		export "$(grep -v '^#' .env | xargs -0)"
	fi

	export APP_DIR=${APP_DIR}

    # NOTE Unlike conf.configu.sh, this script does not 
    # unset env vars subsequently removed from the config file.
    # To refresh your env start a new terminal session

	# Print application env
	printenv | sort | grep --color -E "APP_|AWS_"
}