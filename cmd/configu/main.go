package main

import (
	"github.com/mozey/config/pkg/cmdconfig"
	"github.com/mozey/logutil"
)

// version has a hard-coded default value
// https://github.com/mozey/config/issues/20
// For custom builds, the version can be overwritten with ldflags, see
// "Golang compile environment variable into binary"
// https://stackoverflow.com/a/47665780/639133
var version string = "v0.17.0"

func main() {
	logutil.SetupLogger(true)

	// For custom flags and commands,
	// see comments in pkg/cmdconfig/main.go
	cmdconfig.Main(version)
}
