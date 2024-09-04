package cmdconfig

import (
	"github.com/pkg/errors"
)

// .............................................................................
// Template funcs

// FileNameConfigGo for config.go
const FileNameConfigGo = "config.go"

// FileNameTemplateGo for template.go
const FileNameTemplateGo = "template.go"

// FileNameFnGo for fn.go
const FileNameFnGo = "fn.go"

// GetTemplate returns the text template for the given file name.
func GetTemplate(fileName string) (s string, err error) {
	if fileName == FileNameConfigGo {
		return templateConfigGo, nil
	}

	if fileName == FileNameTemplateGo {
		return templateTemplateGo, nil
	}

	if fileName == FileNameFnGo {
		return templateFnGo, nil
	}

	return s, errors.Errorf("invalid file name %s", fileName)
}

// .............................................................................
// Template strings

// templateConfigGo text template to generate FileNameConfigGo.
// NOTE the "standard header" for recognizing machine-generated files
// https://github.com/golang/go/issues/13560#issuecomment-276866852
var templateConfigGo = `
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

{{range .Keys}}
// {{.KeyPrefix}}
var {{.KeyPrivate}} string{{end}}

// Config fields correspond to config file keys less the prefix
type Config struct {
	{{range .Keys}}
	{{.KeyPrivate}} string // {{.KeyPrefix}}{{end}}
}

{{range .Keys}}
// {{.Key}} is {{.KeyPrefix}}
func (c *Config) {{.Key}}() string {
	return c.{{.KeyPrivate}}
}{{end}}

{{range .Keys}}
// Set{{.Key}} overrides the value of {{.KeyPrivate}}
func (c *Config) Set{{.Key}}(v string) {
	c.{{.KeyPrivate}} = v
}
{{end}}

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
	{{range .Keys}}
	if {{.KeyPrivate}} != "" {
		conf.{{.KeyPrivate}} = {{.KeyPrivate}}
	}
	{{end}}
}

// SetEnv sets non-empty env vars on Config
func SetEnv(conf *Config) {
	var v string

	{{range .Keys}}
	v = os.Getenv("{{.KeyPrefix}}")
	if v != "" {
		conf.{{.KeyPrivate}} = v
	}
	{{end}}
}

// GetMap of all env vars
func (c *Config) GetMap() map[string]string {
	m := make(map[string]string)
	{{range .Keys}}
	m["{{.KeyPrefix}}"] = c.{{.KeyPrivate}}
	{{end}}
	return m
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

	var filePath string
	filePaths, err := share.GetConfigFilePaths(appDir, env)
	if err != nil {
		return conf, err
	}
	for _, filePath = range filePaths {
		_, err := os.Stat(filePath)
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
	if filePath == "" {
		return conf, errors.Errorf("config file not found in %s", appDir)
	}

	b, err := os.ReadFile(filePath)
	if err != nil {
		return conf, errors.WithStack(err)
	}

	configMap := make(map[string]string)
	err = json.Unmarshal(b, &configMap)
	if err != nil {
		return conf, errors.WithStack(err)
	}
	for key, val := range configMap {
		_ = os.Setenv(key, val)
	}
	return New(), nil
}
`

// templateTemplateGo text template to generate FileNameTemplateGo
var templateTemplateGo = `
// Code generated with https://github.com/mozey/config DO NOT EDIT

package config

import (
	"bytes"
	"text/template"
)

{{range .TemplateKeys}}
// Exec{{.Key}} fills {{.KeyPrefix}} with the given params
func (c *Config) Exec{{.Key}}({{.ExplicitParams}}) string {
	t := template.Must(template.New("{{.KeyPrivate}}").Parse(c.{{.KeyPrivate}}))
	b := bytes.Buffer{}
	_ = t.Execute(&b, map[string]interface{}{
	{{range .Params}}
		"{{.Key}}": {{if .Implicit}}c.{{end}}{{.KeyPrivate}},{{end}}
	})
	return b.String()
}
{{end}}
`

// templateFnGo text template to generate FileNameFnGo
var templateFnGo = `
// Code generated with https://github.com/mozey/config DO NOT EDIT

package config

import (
	"fmt"
	"strconv"
	"strings"
)

type Fn struct {
	input string
	// output of the last function,
	// might be useful when chaining multiple functions?
	output string
}

// .............................................................................
// Methods to set function input

{{range .Keys}}
// Fn{{.Key}} sets the function input to the value of {{.KeyPrefix}}
func (c *Config) Fn{{.Key}}() *Fn {
	fn := Fn{}
	fn.input = c.{{.KeyPrivate}}
	fn.output = ""
	return &fn
}
{{end}}

// .............................................................................
// Type conversion functions

// Bool parses a bool from the value or returns an error.
// Valid values are "1", "0", "true", or "false".
// The value is not case-sensitive
func (fn *Fn) Bool() (bool, error) {
	v := strings.ToLower(fn.input)
	if v == "1" || v == "true" {
		return true, nil
	}
	if v == "0" || v == "false" {
		return false, nil
	}
	return false, fmt.Errorf("invalid value %s", fn.input)
}

// Float64 parses a float64 from the value or returns an error
func (fn *Fn) Float64() (float64, error) {
	f, err := strconv.ParseFloat(fn.input, 64)
	if err != nil {
		return f, err
	}
	return f, nil
}

// Int64 parses an int64 from the value or returns an error
func (fn *Fn) Int64() (int64, error) {
	i, err := strconv.ParseInt(fn.input, 10, 64)
	if err != nil {
		return i, err
	}
	return i, nil
}

// String returns the input as is
func (fn *Fn) String() string {
	return fn.input
}
`
