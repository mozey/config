
// Code generated with https://github.com/mozey/config DO NOT EDIT

package config

import (
	"encoding/base64"
	"encoding/json"
	"os"

	"github.com/mozey/config/pkg/share"
	"github.com/pkg/errors"
)

// KeyPrefix is not made publicly available on this package,
// users must use the getter or setter methods.
// This package must not change the config file


// APP_BAR
var bar string
// APP_BUZ
var buz string
// APP_FOO
var foo string
// APP_TEMPLATE_FIZ
var templateFiz string
// APP_DIR
var dir string

// Config fields correspond to config file keys less the prefix
type Config struct {
	
	bar string // APP_BAR
	buz string // APP_BUZ
	foo string // APP_FOO
	templateFiz string // APP_TEMPLATE_FIZ
	dir string // APP_DIR
}


// Bar is APP_BAR
func (c *Config) Bar() string {
	return c.bar
}
// Buz is APP_BUZ
func (c *Config) Buz() string {
	return c.buz
}
// Foo is APP_FOO
func (c *Config) Foo() string {
	return c.foo
}
// TemplateFiz is APP_TEMPLATE_FIZ
func (c *Config) TemplateFiz() string {
	return c.templateFiz
}
// Dir is APP_DIR
func (c *Config) Dir() string {
	return c.dir
}


// SetBar overrides the value of bar
func (c *Config) SetBar(v string) {
	c.bar = v
}

// SetBuz overrides the value of buz
func (c *Config) SetBuz(v string) {
	c.buz = v
}

// SetFoo overrides the value of foo
func (c *Config) SetFoo(v string) {
	c.foo = v
}

// SetTemplateFiz overrides the value of templateFiz
func (c *Config) SetTemplateFiz(v string) {
	c.templateFiz = v
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
	
	if buz != "" {
		conf.buz = buz
	}
	
	if foo != "" {
		conf.foo = foo
	}
	
	if templateFiz != "" {
		conf.templateFiz = templateFiz
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
	
	v = os.Getenv("APP_BUZ")
	if v != "" {
		conf.buz = v
	}
	
	v = os.Getenv("APP_FOO")
	if v != "" {
		conf.foo = v
	}
	
	v = os.Getenv("APP_TEMPLATE_FIZ")
	if v != "" {
		conf.templateFiz = v
	}
	
	v = os.Getenv("APP_DIR")
	if v != "" {
		conf.dir = v
	}
	
}

// GetMap of all env vars
func (c *Config) GetMap() map[string]string {
	m := make(map[string]string)
	
	m["APP_BAR"] = c.bar
	
	m["APP_BUZ"] = c.buz
	
	m["APP_FOO"] = c.foo
	
	m["APP_TEMPLATE_FIZ"] = c.templateFiz
	
	m["APP_DIR"] = c.dir
	
	return m
}

// LoadMap sets the env from a map and returns a new instance of Config
func LoadMap(configMap map[string]string) (conf *Config)  {
	for key, val := range configMap {
		_ = os.Setenv(key, val)
	}
	return New()
}

// SetEnvBase64 decodes and sets env from the given base64 string
func SetEnvBase64(configBase64 string) (err error) {
	// Decode base64
	decoded, err := base64.StdEncoding.DecodeString(configBase64)
	if err != nil {
		return errors.WithStack(err)
	}
	// UnMarshall json
	configMap := make(map[string]string)
	err = json.Unmarshal(decoded, &configMap)
	if err != nil {
		return errors.WithStack(err)
	}
	// Set config
	for key, value := range configMap {
		err = os.Setenv(key, value)
		if err != nil {
			return errors.WithStack(err)
		}
	}

	return nil
}

// LoadFile sets the env from file and returns a new instance of Config
func LoadFile(env string) (conf *Config, err error) {
	appDir := os.Getenv("APP_DIR")
	if appDir == "" {
		// Use current working dir
		appDir, err = os.Getwd()
		if err != nil {
			return conf, errors.WithStack(err)
		}
	}

	var configPath string
	filePaths, err := share.GetConfigFilePaths(appDir, env)
	if err != nil {
		return conf, err
	}
	for _, configPath = range filePaths {
		_, err := os.Stat(configPath)
		if err != nil {
			if os.IsNotExist(err) {
				// Path does not exist
				continue
			}
			return conf, errors.WithStack(err)
		}
		// Path exists
		break
	}
	if configPath == "" {
		return conf, errors.Errorf("config file not found in %s", appDir)
	}

	b, err := os.ReadFile(configPath)
	if err != nil {
		return conf, errors.WithStack(err)
	}

	configMap, err := share.UnmarshalConfig(configPath, b)
	if err != nil {
		return conf, err
	}
	for key, val := range configMap {
		_ = os.Setenv(key, val)
	}
	return New(), nil
}
