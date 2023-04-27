# [mozey/config](https://github.com/mozey/config)

Manage env vars with a flat key/value file. See [architecture notes](https://github.com/mozey/config#architecture-notes) for more info.

By default `${ENV} == "dev"`, and your config file may be named **.env**. 

The convention is to add the *"config file"* to **.gitignore**, and a *"sample config file"*, **sample.env**, is versioned with your code.

For multiple environments, e.g. `${ENV} == "prod"`, the *"config file naming convention"* is **${ENV}.env**, that is **prod.env**. 

Other files types, like JSON, are also supported. The naming convention then is to add a prefix: **config.${ENV}.json**.

List of features
- [Command](https://github.com/mozey/config#quick-start) to manage the env: `configu`
- [Bash](https://www.gnu.org/software/bash/) function to [toggle env](https://github.com/mozey/config#toggling-env): `conf`
- [Generate code](https://github.com/mozey/config#generate-config-package) (e.g. `pkg/config/config.go`) to include in your [Go module](https://go.dev/blog/using-go-modules)
- And more...

File loading precedence for `configu` command (default `${ENV} == "dev"`)
- config.dev.json
- config.json
- dev.env
- .env
- config.dev.yaml
- config.yaml


## Quick start

Install from source

**WARNING** Do not run the command below inside a clone of this repo,
or inside any folder this is a *"go module"*, i.e. has a **go.mod** file
```bash
# Since Go 1.20.3 
# "'go get' is no longer supported outside a module"
go install github.com/mozey/config/cmd/configu@latest
```

Create a config file
```bash
echo '{"APP_FOO": "foo", "APP_BAR": "foo"}' > config.dev.json

# This env var will be removed,
# it is not listed in the config file
export APP_VAR_NOT_IN_CONFIG_FILE="not_for_this_app"

printenv | sort | grep --color -E "APP_"
```

Print commands
```bash
export APP_DIR=$(pwd)
${GOPATH}/bin/configu
```

Reset env
```bash
eval "$(${GOPATH}/bin/configu)"

printenv | sort | grep --color -E "APP_"
```

Set a key value pair in `config.dev.json`
```bash
${GOPATH}/bin/configu -key APP_FOO -value xxx
```

Set a key value pair for all `config.*.json` and `sample.config.*.json` files in APP_DIR
```bash
${GOPATH}/bin/configu -all -key APP_FOO -value xxx
```

Convert config file to a different format
```bash
# dev.env
${GOPATH}/bin/configu -format env
# If you require only a single config file
mv dev.env .env

# config.dev.yaml
${GOPATH}/bin/configu -format yaml
```


## Toggling env

This repo includes a [conf.sh](https://github.com/mozey/config/blob/master/conf.sh) script that makes use of built-in Bash commands. The script creates a `conf` Bash function to set environment variables. Only **.env** files are supported

Download the script to your home dir
```bash
curl https://raw.githubusercontent.com/mozey/config/master/conf.sh --output ${HOME}/.conf.sh
```

Source the script on [bash startup](https://www.gnu.org/software/bash/manual/html_node/Bash-Startup-Files.html), e.g. `~/.bashrc`, to create the conf func
```bash
source ${HOME}/.conf.sh
```

Use the func to toggle env
```bash
conf

conf prod

# Tip: don't create a prod config file on your dev machine!
conf stage
```

The default script for toggling env does not make use of the `configu` command. This facilitates using the workflow described in this repo on systems where the command is not installed. See [toggling env with configu](https://github.com/mozey/config#toggling-env-with-configu).


## Toggling env with configu

This repo also includes a [conf.configu.sh](https://github.com/mozey/config/blob/master/conf.configu.sh) script that makes use of the `configu` command. The script creates a `conf` Bash function to set **and unset** environment variables.

Setup instruction as the same as for [toggling env](https://github.com/mozey/config#toggling-env), except that a different file must be downloaded
```bash
curl https://raw.githubusercontent.com/mozey/config/master/conf.configu.sh --output ${HOME}/.conf.sh
```


## Generate config package

The config package can be included in your app. It is useful for
- completion
- reading and setting env
- load config from file for testing

Use the `-dry-run` flag to print the result and skip the update
```bash
configu -generate pkg/config -dry-run
```

Refresh the package after adding or removing config keys
```bash
mkdir -p pkg/config

configu -generate pkg/config

go fmt ./pkg/config/config.go
```


## Build script

Duplicate `scripts/config.sh` in your module.

Use the config script to
- create the dev config file
- generate the config package

```bash
APP_DIR=$(pwd) ./scripts/config.sh
```


## Prod env

The `configu` cmd uses `config.dev.json` by default.

Create `config.prod.json` and set a key
```bash
cp ./sample.config.prod.json ./config.prod.json

configu -env prod -key APP_BEER -value pilsner
```

Export `prod` env
```bash
configu -env prod

eval "$(configu -env prod)"

printenv | sort | grep -E "APP_"
```

### Compare config files and print un-matched keys

It's advisable for all config files to have the same keys, if a key does not apply to an env then set the value to an empty string. See [architecture notes](https://github.com/mozey/config#architecture-notes).

Comparing keys
```bash
configu -env dev -compare prod

# cmd exits with error code if the keys don't match
echo $?
```

Compare keys in `config.dev.json` with `sample.config.dev.json`
```bash
configu -env dev -compare sample.dev
```

Set a key value in `config.prod.json`.
```bash
./configu -env prod -key APP_FOO -value xxx
```


## Dev setup

Get the code
```bash
git clone https://github.com/mozey/config.git

cd config
```

Reset to remove ignored files
```bash
APP_DIR=$(pwd) ./scripts/reset.sh
```

`APP_DIR` must be always be set to the module root. All config env vars must have a prefix, the default is `APP_`

Run the tests
```bash
APP_DIR=$(pwd) gotest -v ./...
```

Compare generated files in `pkg/cmdconfig/testdata` to `pkg/cmdconfig/testdata/compare`

Update testdata if required (after adding new features)
```bash
configu -generate pkg/config
cp pkg/config/config.go pkg/cmdconfig/testdata/config.go
cp sample.config.dev.json pkg/cmdconfig/testdata/config.dev.json
```


## Debug

Create `config.dev.json`
```bash
cp ./sample.config.dev.json ./config.dev.json
```

Run the `configu` cmd. By default it reads `config.dev.json`, and prints the key/value pairs formatted as commands to export env vars
```bash
APP_DIR=$(pwd) go run cmd/configu/main.go
```

Build from source
```bash
go build -o ${GOPATH}/bin/configu ./cmd/configu
```


## Advanced Usage

Duplicate `cmd/configu/main.go` in your module.

The `configu` command can be customized,
see comments in the `config.Main` func

Build the `configu` command.
The APP_DIR env var is required
```bash
export APP_DIR=$(pwd)

go build -o ${APP_DIR}/configu ./cmd/configu
```

Then use the local command
```bash
./configu
```


## Windows

Installation work the same thanks to Go.

**WARNING** Do not run the command below inside a clone of this repo,
or inside any folder this is a "go module", i.e. has a `go.mod` file,
otherwise the install (or update to the latest tag) won't work
```bash
# Since Go 1.20.3 
# "'go get' is no longer supported outside a module"
go install github.com/mozey/config/cmd/configu@latest
```

Depends on [clink](https://mridgers.github.io/clink) and [gow](https://github.com/bmatzelle/gow/wiki), install them first. Then the following commands can be executed in the command prompt `cmd.exe`

Create a config file
```bat
echo {"APP_FOO": "foo", "APP_BAR": "foo"} > config.dev.json

REM This env var will be removed,
REM it is not listed in the config file
set APP_VAR_NOT_IN_CONFIG_FILE="not_for_this_app"

printenv | grep APP_
```

Print commands
```bat
set APP_DIR=%cd%
%GOPATH%/bin/configu
```

Reset env
```bat
eval "$(${GOPATH}/bin/configu)"

printenv | grep APP_
```

Set a key value pair in **config.dev.json**
```bat
%GOPATH%/bin/configu -key APP_FOO -value xxx
```

Toggle config, first update PATH to make `conf.bat` available
```bat
REM Right click start - System - Advanced system settings - Advanced - Environment Variables...
REM %GOPATH%/src/github.com/mozey/config

conf.bat
```

Run the tests
```bat
set APP_DIR=%cd%
gotest -v ./...
```


## Architecture notes

*The twelve-factor app stores config in environment variables*, i.e. 
[read config from the environment](https://12factor.net/config).

Nested config is not supported, the config file must have a flat key value structure: *"env vars are granular controls, each fully orthogonal to other env vars. They are never grouped together..."*. 

*"Keep development, staging, and production as similar as possible"*, see [dev/prod parity](https://12factor.net/dev-prod-parity). See [compare config files and print un-matched keys](https://github.com/mozey/config#compare-config-files-and-print-un-matched-keys)

[Env vars](https://en.wikipedia.org/wiki/Environment_variable) 
must be set in the parent process. **Apps must not set their own config, they read it from the environment.**. An exception to this rule is [base64 config](https://github.com/mozey/config/issues/28) that is compiled into, and distributed with binaries.

Above is in contrast to using [Viper](https://github.com/spf13/viper) for [reading config files from your application](https://github.com/spf13/viper#reading-config-files). In addition, while [Viper has the ability to bind to flags](https://github.com/spf13/viper#working-with-flags), this repo encourages using the [standard flag package](https://pkg.go.dev/flag). Or if you prefer, Viper can be used in combination with this repo. In short, Viper is very flexible, while this repo is more opinionated.

[Notes re. twelve factor apps](https://github.com/mozey/config/issues/5)


## Key naming conventions

All keys must start with the **same prefix**.

Keys are case sensitive, and it's advised to make keys **all uppercase**.

Use (uppercase) **SNAKE_CASE**.

Assuming the default key prefix `APP_`, to avoid un-defined behaviour when generating package code, **do not start keys with**
- `APP_EXEC_TEMPLATE_`
- `APP_FN_`
- `APP_SET_`
See [error on reserved prefix when generating package code](https://github.com/mozey/config/issues/37)

In addition to the `APP_` prefix, the configu command also supports additional prefixes like `AWS_`.

The `APP_DIR` key is set to the working directory when toggling env, any value specified for this key in the config file will be overridden
