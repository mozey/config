package main

import (
	"github.com/mozey/config/pkg/cmdconfig"
	"github.com/mozey/logutil"
	"runtime"
)

func main() {
	if runtime.GOOS == "window" {
		// TODO Console writer bad characters on Windows
		logutil.SetupLogger(false)
	} else {
		logutil.SetupLogger(true)
	}

	// For custom flags and commands,
	// see comments in config.Main...
	cmdconfig.Main()
}
