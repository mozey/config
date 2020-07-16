
// Code generated with https://github.com/mozey/config DO NOT EDIT

package config

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
)


// APP_BAR
var bar string
// APP_FOO
var foo string
// APP_DIR
var dir string


// Config fields correspond to config file keys less the prefix
type Config struct {

	bar string // APP_BAR
	foo string // APP_FOO
	dir string // APP_DIR
}


// Bar is APP_BAR
func (c *Config) Bar() string {
	return c.bar
}
// Foo is APP_FOO
func (c *Config) Foo() string {
	return c.foo
}
// Dir is APP_DIR
func (c *Config) Dir() string {
	return c.dir
}


// SetBar overrides the value of bar
func (c *Config) SetBar(v string) {
	c.bar = v
}

// SetFoo overrides the value of foo
func (c *Config) SetFoo(v string) {
	c.foo = v
}

// SetDir overrides the value of dir
func (c *Config) SetDir(v string) {
	c.dir = v
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

	if foo != "" {
		conf.foo = foo
	}

	if dir != "" {
		conf.dir = dir
	}

}

// SetEnv sets non-empty env vars on Config
func SetEnv(conf *Config) {
	var v string


	v = os.Getenv("APP_BAR")
	if v != "" {
		conf.bar = v
	}

	v = os.Getenv("APP_FOO")
	if v != "" {
		conf.foo = v
	}

	v = os.Getenv("APP_DIR")
	if v != "" {
		conf.dir = v
	}

}

// LoadFile sets the env from file and returns a new instance of Config
func LoadFile(mode string) (conf *Config, err error) {
	appDir := os.Getenv("APP_DIR")
	p := fmt.Sprintf("%v/config.%v.json", appDir, mode)
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
		_ = os.Setenv(key, val)
	}
	return New(), nil
}
