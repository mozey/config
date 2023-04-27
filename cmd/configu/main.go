package main

import (
	"github.com/mozey/config/pkg/cmdconfig"
	"github.com/mozey/logutil"
)

func main() {
	logutil.SetupLogger(true)

	// For custom flags and commands,
	// see comments in pkg/cmdconfig/main.go
	cmdconfig.Main()
}
