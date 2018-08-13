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
    
Create `config.json`

    cd ${APP_DIR}
    
    ./config \
    -key APP_DIR -value ${APP_DIR} \
    -update
    
Set env from config

    eval "$(./config)"
    export APP_FOO=unset_this
    printenv | sort | grep -E 'APP_'
    
Unset env with `APP_` prefix not listed in config
    
    eval "$(./config)"
    printenv | sort | grep -E 'APP_'


# Testing

    cd ${GOPATH}/src/github.com/mozey/config

    export APP_DEBUG=true
    gotest -v ./cmd/config/... -run TestPrintEnvCommands
    
    gotest ./cmd/config/...
    