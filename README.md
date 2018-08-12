# config

Manage env vars with a config.json file


# Quick start

    go get github.com/mozey/config

Set `APP_DIR` 

    export APP_DIR=${GOPATH}/src/your/project/path
    
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
    
Set env

    $(./config)
    
    printenv | sort | grep -E 'APP_'
