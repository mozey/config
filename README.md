# config

Manage env vars with a flat config.json file

`mozey/config` has the following components
- Command to manage the env: `configu`
- Bash function to toggle env: `conf`
- Generate a package (e.g. `pkg/config/config.go`) to include in your module


## Quick start

Install pre-compiled, download appropriate link below
- [Linux 386](https://github.com/mozey/config/releases/download/v0.3.3/configu) 
- [Windows 386](https://github.com/mozey/config/releases/download/v0.3.3/configu.exe)

Install from source

    go get -u github.com/mozey/config/...
    
Create a config file
    
    echo '{"APP_FOO": "foo", "APP_BAR": "foo"}' > config.dev.json

    # This env var will be removed,
    # it is not listed in the config file 
    export APP_VAR_NOT_IN_CONFIG_FILE="not_for_this_app"
    
    printenv | grep APP_
    
Print commands

    export APP_DIR=$(pwd)
    ${GOPATH}/bin/configu
    
Reset env

    eval "$(${GOPATH}/bin/configu)"

    printenv | grep APP_
    
Set a key value in `config.dev.json`

    ${GOPATH}/bin/configu -key APP_FOO -value xxx
    
    
## Toggling env

Copy the conf script to your home dir

    cp ./conf.sh ~/.conf.sh

Source the script on [bash startup](https://www.gnu.org/software/bash/manual/html_node/Bash-Startup-Files.html),
e.g. `~/.bashrc`, to create the conf func

    source ${HOME}/.conf.sh
    
Use the func to toggle env

    conf 
    
    conf prod
    
    # Tip: don't create a prod config file on your dev machine! 
    conf stage
    
    
## Generate config package

The config package can be included in your app. It is useful for 
- completion
- reading and setting env
- load config from file for testing

Use the `-dry-run` flag to print the result and skip the update

    configu -generate pkg/config -dry-run
    
Refresh the package after adding or removing config keys

    mkdir -p pkg/config
    
    configu -generate pkg/config
    
    go fmt ./pkg/config/config.go

    
## Build script

Duplicate `scripts/config.sh` in your module.

Use the config script to
- create the dev config file
- generate the config package
    

    APP_DIR=$(pwd) ./scripts/config.sh


## Prod env

The `configu` cmd uses `config.dev.json` by default.

Create `config.prod.json` and set a key

    cp ./config.prod.sample.json ./config.prod.json
    
    configu -env prod -key APP_BEER -value pilsner
    
Export `prod` env

    configu -env prod
    
    eval "$(configu -env prod)"
    
    printenv | sort | grep -E "APP_"
    
All config files must have the same keys,
if a key is n/a in for an env then set the value to an empty string.
Compare config files and print un-matched keys

    configu -env dev -compare prod
    
    # cmd exits with error code if the keys don't match
    echo $?
    
Compare keys in `config.dev.json` with `sample.config.dev.json`

    configu -env dev -compare sample.dev
    
Set a key value in `config.prod.json`.

    ./configu -env prod -key APP_FOO -value xxx
    

## Dev setup

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
    
Run the `configu` cmd.
By default, echo `config.dev.json`,
formatted as commands to export env vars

    APP_DIR=$(pwd) go run cmd/configu/main.go
    
    
## Advanced Usage

Duplicate `cmd/configu/main.go` in your module.

The `configu` command can be customized,
see comments in the `config.Main` func

Build the `configu` command.
The APP_DIR env var is required

    export APP_DIR=$(pwd) 
    
    go build -o ${APP_DIR}/configu ./cmd/configu
    
Then use the local command

    ./configu 
    
    
## Windows

Install

    go get -u github.com/mozey/config/...
    
Depends on [clink](https://mridgers.github.io/clink) and
[gow](https://github.com/bmatzelle/gow/wiki), install them first.
Then the following commands can be executed in `cmd.exe`

Create a config file
    
    echo {"APP_FOO": "foo", "APP_BAR": "foo"} > config.dev.json

    # This env var will be removed,
    # it is not listed in the config file 
    set APP_VAR_NOT_IN_CONFIG_FILE="not_for_this_app"
    
    printenv | grep APP_
    
Print commands

    set APP_DIR=%cd%
    %GOPATH%/bin/configu
    
Reset env

    eval "$(${GOPATH}/bin/configu)"

    printenv | grep APP_
    
Set a key value in `config.dev.json`

    %GOPATH%/bin/configu -key APP_FOO -value xxx
    
Toggle config, first update PATH to make `conf.bat` available 

    Right click start > System > Advanced system settings > Advanced > Environment Variables... 
    %GOPATH%/src/github.com/mozey/config
    
    conf.bat
    
Run the tests

    set APP_DIR=%cd% 
    gotest -v ./...    


## TODO [Viper](https://github.com/spf13/viper) 

Does it make sense to build this on top of, or use Viper instead?

How would the config package be generated?

Keep in mind that env must be set in the parent process.
E.g. apps should not set their own config, they must read it from the env 


