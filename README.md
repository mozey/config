# config

Manage env vars with a flat config.json file

It has two components
- module specific `config` command to manage the env,
- a generated config helper file to include in your module


## Dev setup

Get the code 

    git clone https://github.com/mozey/config.git
    
Reset to remove ignored files

    cd config
    ./reset.sh
    
`APP_DIR` must be always be set to the module root. 
All config env vars must have a prefix, the default is `APP_`

Run the tests

    APP_DIR=$(pwd) gotest -v ./...
    
    
## Debug

Create `config.dev.json`
                        
    export APP_DIR=$(pwd) 
    cp ${APP_DIR}/config.dev.sample.json ${APP_DIR}/config.dev.json
    
Run the `config` cmd.
By default echo `config.dev.json`,
formatted as commands to export env vars

    go run -ldflags "-X main.AppDir=${APP_DIR}" cmd/config/main.go
    
    
## Basic Usage

Duplicate `cmd/config/main.go` in your module

Build the `config` command

    APP_DIR=$(pwd) go build \
    -ldflags "-X main.AppDir=${APP_DIR}" \
    -o ${APP_DIR}/config ./cmd/config 

...and use it to set a key value in `config.dev.json`.
Note that `APP_DIR` is also set if missing

    ./config -key APP_FOO -value xxx

...or manage the env vars in your shell

    # This env var will be removed,
    # it is not listed in the config file 
    export APP_VAR_NOT_IN_CONFIG_FILE="not_for_this_app" 
    
    # Print commands
    ./config

    # Set env    
    eval "$(./config)"
    
    # Print env
    printenv | sort | grep -E "APP_"
 
    
## Prod env

The `config` cmd uses `config.dev.json` by default.

Create `config.prod.json` and set a key

    cp ${APP_DIR}/config.prod.sample.json ${APP_DIR}/config.prod.json
    
    ./config -env prod -key APP_BEER -value pilsner
    
Export `prod` env

    ./config -env prod 
    
    eval "$(./config -env prod)"
    
    printenv | sort | grep -E "APP_"
    
All config files must have the same keys,
if a key is n/a in for an env then set the value to an empty string.
Compare config files and print un-matched keys

    ./config -env dev -compare prod
    
    # cmd exits with error code if the keys don't match
    echo $?


## Generate config helper

The config helper can be included in your app. It is useful for 
- completion
- reading and setting env
- load config from file for testing

Refresh the helper after adding or removing config keys

    mkdir -p pkg/example
    
    ./config -generate pkg/example
    
    go fmt ./pkg/example/config.go

Use the `-dry-run` flag to print the result and skip the update

    ./config -generate pkg/example -dry-run


# Toggling env

Create the func below to in your bash profile to quickly toggle env

    # Helper func to toggle env with github/mozey/config
    conf() {
        local ENV=${1}
        if [[ -z "${ENV}" ]]; then
            local ENV="dev"
        fi
    
        if ! test -f "./config"; then
            echo "config cmd not found"
            return 1
        fi
    
        if test -f "./config.${ENV}.json"; then
            eval "$(./config -env ${ENV})"
            # Checking retVal with $? won't work here
            printenv | sort | grep -E "APP_|AWS_"
        else
            echo "config file not found"
            return 1
        fi
    }
    
Then use it to toggle env

    conf 
    
    conf prod
    
    # Tip: don't create a prod config file on your dev machine! 
    conf stage
    

# TODO [Viper](https://github.com/spf13/viper) 

Does it make sense to build this on top of, or use Viper instead?

How would the config helper be generated?

Keep in mind that env must be set in the parent process.
E.g. apps should not set their own config, they must read it from the env 


