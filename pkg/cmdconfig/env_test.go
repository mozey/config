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
	
	# quotes are included in the value
	APP_FOO="foo"
	
	APP_TEMPLATE = my name is {{.Name}}
	
	AWS_PROFILE=aws-local
	`)

	m, err := UnmarshalENV(envFileBytes)
	is.NoErr(err)
	is.Equal("\"foo\"", m["APP_FOO"])
	is.Equal("my name is {{.Name}}", m["APP_TEMPLATE"])
	is.Equal("aws-local", m["AWS_PROFILE"])
}

func TestMarshalENV(t *testing.T) {
	is := testutil.Setup(t)

	m := make(map[string]string)
	m["APP_FOO"] = "\"foo\""
	m["APP_TEMPLATE"] = "my name is {{.Name}}"
	m["AWS_PROFILE"] = "aws-local"

	b, err := MarshalENV(m)
	is.NoErr(err)

	// TODO Preserve shebang line?
	envFileBytes := []byte(`APP_FOO="foo"
APP_TEMPLATE=my name is {{.Name}}
AWS_PROFILE=aws-local
`)

	is.Equal(envFileBytes, b)
}
