
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

// FnBar sets the function input to the value of APP_BAR
func (c *Config) FnBar() *Fn {
	fn := Fn{}
	fn.input = c.bar
	fn.output = ""
	return &fn
}

// FnBuz sets the function input to the value of APP_BUZ
func (c *Config) FnBuz() *Fn {
	fn := Fn{}
	fn.input = c.buz
	fn.output = ""
	return &fn
}

// FnFoo sets the function input to the value of APP_FOO
func (c *Config) FnFoo() *Fn {
	fn := Fn{}
	fn.input = c.foo
	fn.output = ""
	return &fn
}

// FnTemplateFiz sets the function input to the value of APP_TEMPLATE_FIZ
func (c *Config) FnTemplateFiz() *Fn {
	fn := Fn{}
	fn.input = c.templateFiz
	fn.output = ""
	return &fn
}

// FnDir sets the function input to the value of APP_DIR
func (c *Config) FnDir() *Fn {
	fn := Fn{}
	fn.input = c.dir
	fn.output = ""
	return &fn
}

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
