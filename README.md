# config

Manage env vars with a flat config.json file

It has two components
- module specific `config` command to manage the env,
- generated helper file (`pkg/config/config.go`) to include in your module

## Quick start

Get the code 

    git clone https://github.com/mozey/config.git
    
    cd config
    
Reset to remove ignored files

    APP_DIR=$(pwd) ./scripts/reset.sh
    
`APP_DIR` must be always be set to the module root. 
All config env vars must have a prefix, the default is `APP_`

Run the tests

    APP_DIR=$(pwd) gotest -v ./...

    
## Debug    

Create `config.dev.json`
                        
    cp ./config.dev.sample.json ./config.dev.json
    
Run the `config` cmd.
By default echo `config.dev.json`,
formatted as commands to export env vars

    APP_DIR=$(pwd) go run cmd/config/main.go
    
    
## Basic Usage

Duplicate `cmd/config/main.go` in your module.

The `config` command can be customized,
see comments in the `config.Main` func

Build the `config` command.
The APP_DIR env var is required

    export APP_DIR=$(pwd) 
    
    go build -o ${APP_DIR}/config ./cmd/config 

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

    cp ./config.prod.sample.json ./config.prod.json
    
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

    mkdir -p pkg/config
    
    ./config -generate pkg/config
    
        go fmt ./pkg/config/config.go

Use the `-dry-run` flag to print the result and skip the update

    ./config -generate pkg/config -dry-run


    ## Toggling env

Copy the conf script to your home dir

    cp ./conf.sh ~/.conf.sh

Source the script on [bash startup](https://www.gnu.org/software/bash/manual/html_node/Bash-Startup-Files.html),
e.g. `~/.bashrc`, to create the conf func

    source ${HOME}/.conf.sh
    
Use the alias to toggle env

    conf 
    
    conf prod
    
    # Tip: don't create a prod config file on your dev machine! 
    conf stage
    
## Build script

Duplicate `scripts/config.sh` in your module.

Use the config script to
- build the config cmd
- create the dev config file
- generate the config helper
    

    APP_DIR=$(pwd) ./scripts/config.sh
    

## TODO [Viper](https://github.com/spf13/viper) 

Does it make sense to build this on top of, or use Viper instead?

How would the config helper be generated?

Keep in mind that env must be set in the parent process.
E.g. apps should not set their own config, they must read it from the env 


