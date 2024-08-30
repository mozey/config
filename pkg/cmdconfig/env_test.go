package cmdconfig

import (
	"testing"

	"github.com/mozey/config/pkg/testutil"
)

func TestUnmarshalENV(t *testing.T) {
	is := testutil.Setup(t)

	envFileBytes := []byte(`
	#!/bin/bash xyz

	Empty lines and lines not matching VAR=VAL are ignored
	rm -rf /

	# This is a comment

	# surrounding quotes trimmed from the value
	APP_FOO="foo"

	# inner quotes are included in the value
	APP_TEMPLATE = "my name is "{{.Name}}""

	AWS_PROFILE=aws-local
	`)

	m, err := UnmarshalENV(envFileBytes)
	is.NoErr(err)
	is.Equal("foo", m["APP_FOO"])
	is.Equal("my name is \"{{.Name}}\"", m["APP_TEMPLATE"])
	is.Equal("aws-local", m["AWS_PROFILE"])
}

func TestMarshalENV(t *testing.T) {
	is := testutil.Setup(t)

	c := &conf{}
	c.Map = make(map[string]string)
	c.Map["APP_FOO"] = "\"foo\""
	c.Map["APP_TEMPLATE"] = "my name is {{.Name}}"
	c.Map["AWS_PROFILE"] = "aws-local"
	c.refreshKeys()

	b, err := MarshalENV(c)
	is.NoErr(err)

	// TODO Preserve comments
	// https://github.com/mozey/config/issues/34
	envFileBytes := []byte(`APP_FOO="foo"
APP_TEMPLATE=my name is {{.Name}}
AWS_PROFILE=aws-local
`)

	is.Equal(string(envFileBytes), string(b))
}
