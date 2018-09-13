// Code generated with https://github.com/mozey/config DO NOT EDIT

package config

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"
)


// APP_BAR
var bar string
// APP_DIR
var dir string
// APP_FOO
var foo string


// Config fields correspond to config file keys less the prefix
type Config struct {

	bar string // APP_BAR
	dir string // APP_DIR
	foo string // APP_FOO
}


// Bar is APP_BAR
func (c *Config) Bar() string {
	return c.bar
}
// Dir is APP_DIR
func (c *Config) Dir() string {
	return c.dir
}
// Foo is APP_FOO
func (c *Config) Foo() string {
	return c.foo
}

// New creates an instance of Config.
// Build with ldflags to set the package vars.
// Env overrides package vars.
// Fields correspond to the config file keys less the prefix.
// The config file must have a flat structure
func New() *Config {
	conf := &Config{}
	SetVars(conf)
	SetEnv(conf)
	return conf
}

// SetVars sets non-empty package vars on Config
func SetVars(conf *Config) {

	if bar != "" {
		conf.bar = bar
	}

	if dir != "" {
		conf.dir = dir
	}

	if foo != "" {
		conf.foo = foo
	}

}

// SetEnv sets non-empty env vars on Config
func SetEnv(conf *Config) {
	var v string


	v = os.Getenv("APP_BAR")
	if v != "" {
		conf.bar = v
	}

	v = os.Getenv("APP_DIR")
	if v != "" {
		conf.dir = v
	}

	v = os.Getenv("APP_FOO")
	if v != "" {
		conf.foo = v
	}

}

// LoadFile sets the env from file and returns a new instance of Config
func LoadFile(mode string) (conf *Config, err error) {
	p := fmt.Sprintf(path.Join(os.Getenv("GOPATH"),
		"/src/github.com/mozey/config/cmd/config/testdata/config.%v.json"), mode)
	b, err := ioutil.ReadFile(p)
	if err != nil {
		return nil, err
	}
	configMap := make(map[string]string)
	err = json.Unmarshal(b, &configMap)
	if err != nil {
		return nil, err
	}
	for key, val := range configMap {
		os.Setenv(key, val)
	}
	return New(), nil
}