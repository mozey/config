package cmdconfig

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUnmarshalENV(t *testing.T) {
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
	require.NoError(t, err)
	require.Equal(t, "\"foo\"", m["APP_FOO"])
	require.Equal(t, "my name is {{.Name}}", m["APP_TEMPLATE"])
	require.Equal(t, "aws-local", m["AWS_PROFILE"])
}

func TestMarshalENV(t *testing.T) {
	m := make(map[string]string)
	m["APP_FOO"] = "\"foo\""
	m["APP_TEMPLATE"] = "my name is {{.Name}}"
	m["AWS_PROFILE"] = "aws-local"

	b, err := MarshalENV(m)
	require.NoError(t, err)

	// TODO Preserve shebang line?
	envFileBytes := []byte(`APP_FOO="foo"
APP_TEMPLATE=my name is {{.Name}}
AWS_PROFILE=aws-local
`)

	require.Equal(t, envFileBytes, b)
}
