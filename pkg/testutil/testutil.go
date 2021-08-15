package testutil

import (
	config "github.com/mozey/config/pkg/cmdconfig/testdata"
)

var conf *config.Config

// Config loads dev config from file once and returns a copy.
// To override variables per test use the setter functions on conf
func Config() (*config.Config, error) {
	var err error
	// If config is not set
	if conf == nil {
		// Load config
		conf, err = config.LoadFile("dev")
		if err != nil {
			return conf, err
		}
	}
	// Copy the struct not the pointer!
	confCopy := *conf
	return &confCopy, nil
}
