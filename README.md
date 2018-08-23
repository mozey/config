# config

Manage env vars with a config.json file


# Quick start

    go get github.com/mozey/config

Set `APP_DIR`, change command below to use your project path

    export APP_DIR=${GOPATH}/src/github.com/mozey/gateway

Compile

    cd ${GOPATH}/src/github.com/mozey/config

    go build \
    -ldflags "-X main.AppDir=${APP_DIR}" \
    -o ${APP_DIR}/config ./cmd/config
    
Create `config.json` and set a key
                        
    cd ${APP_DIR}
    
    touch config.dev.json
    
    ./config \
    -key APP_DIR -value ${APP_DIR} \
    -update
    
    cat config.dev.json
    
Set env from config

    eval "$(./config)"
    export APP_FOO=unset_this
    printenv | sort | grep -E "APP_"
    
Unset env with `APP_` prefix not listed in config
    
    eval "$(./config)"
    printenv | sort | grep -E "APP_"
    
Generate config helper,
keys in dev must be a subset of prod

    ./config -env prod -gen internal/config
    
    go fmt ./internal/config/config.go
    
    
# Testing

    cd ${GOPATH}/src/github.com/mozey/config

    export APP_DEBUG=true
    gotest -v ./cmd/config/... -run TestPrintEnvCommands
    
    gotest ./cmd/config/...
    
    
# Prod env

Create `config.json` and set a key

    cd ${APP_DIR}
    
    touch config.prod.json
    
    ./config -env prod \
    -key APP_PROD -value true \
    -update
    
    cat config.prod.json
    

# Aliases

Create aliases to toggle between env

    alias dev='eval "$(./config)" && printenv | sort | grep -E "APP_|"'
    alias prod='eval "$(./config -env prod)" && printenv | sort | grep -E "APP_|"'


