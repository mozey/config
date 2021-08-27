package cmdconfig

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	config "github.com/mozey/config/pkg/cmdconfig/testdata"
	"github.com/stretchr/testify/require"
)

func stripGenerated(generated string) string {
	generated = strings.Replace(generated, " ", "", -1)
	generated = strings.Replace(generated, "\t", "", -1)
	generated = strings.Replace(generated, "\n", "", -1)
	return generated
}

func TestGenerateHelpersPrint(t *testing.T) {
	var err error

	appDir := os.Getenv("APP_DIR")
	require.NotEmpty(t, appDir, "APP_DIR must not be empty")

	in := &CmdIn{}
	in.DryRun = true // Do not write files to disk
	in.Prefix = "APP_"
	in.Env = "dev"

	// Path to generate config helpers is not used since dry run is set.
	// Compare with TestGenerateHelpers
	in.Generate = filepath.Join("pkg", "cmdconfig", "testdata")

	in.AppDir = filepath.Join(appDir, in.Generate)

	// Use pkg/cmdconfig/testdata/config.dev.json
	// See "Test fixtures in Go"
	// https://dave.cheney.net/2016/05/10/test-fixtures-in-go
	_, in.Config, err = NewConfig("testdata", in.Env)
	require.NoError(t, err)

	out, err := Cmd(in)
	require.NoError(t, err)
	require.Equal(t, CmdGenerate, out.Cmd)
	require.Equal(t, 0, out.ExitCode)
	require.Equal(t, 3, len(out.Files),
		"Unexpected number of files")

	for _, file := range out.Files {
		fileName := filepath.Base(file.Path)
		generated := stripGenerated(file.Buf.String())
		b, err := ioutil.ReadFile(filepath.Join("testdata", fileName))
		require.NoError(t, err)
		ref := string(b)
		ref = stripGenerated(ref)
		require.Equal(t, ref, generated,
			fmt.Sprintf(
				"generated should match pkg/cmdconfig/testdata/%s", fileName))
	}

	// We've checked the generated code match the files in pkg/cmdconfig/testdata,
	// now check the generated code works as expected...
	err = os.Setenv("APP_DIR", filepath.Join(appDir, in.Generate))
	require.NoError(t, err)
	c, err := config.LoadFile("dev")
	require.NoError(t, err)
	err = os.Setenv("APP_DIR", appDir)
	require.NoError(t, err)
	require.Equal(t, "foo", c.Foo())
	require.Equal(t, "bar", c.Bar())
	require.Equal(t, "Buzz", c.Buz())
	require.Equal(t, "FizzBuzz-FizzBuzz", c.ExecTemplateFiz("-FizzBuzz"))
}

// TestGenerateHelpersSave also covers Files_Save
func TestGenerateHelpersSave(t *testing.T) {
	tmp, err := ioutil.TempDir("", "mozey-config")
	require.NoError(t, err)
	defer (func() {
		_ = os.RemoveAll(tmp)
	})()

	in := &CmdIn{}
	in.AppDir = tmp
	in.DryRun = false // Test writing files to disk
	in.Prefix = "APP_"
	in.Env = "dev"

	// Convention is to keep the helpers in YOUR_PROJECTS_APP_DIR/pkg/config
	in.Generate = filepath.Join("pkg", "config")

	// Use pkg/cmdconfig/testdata/config.dev.json
	_, in.Config, err = NewConfig("testdata", in.Env)
	require.NoError(t, err)

	out, err := Cmd(in)
	require.NoError(t, err)
	require.Equal(t, CmdGenerate, out.Cmd)
	require.Equal(t, 0, out.ExitCode)
	require.Equal(t, 3, len(out.Files),
		"Unexpected number of files")

	// Write the files
	// TODO in.Process call fmt.Println,
	// temporarily capture stdout to avoid cluttering test output?
	// See https://github.com/mozey/go-capturer
	exitCode, err := in.Process(out)
	require.NoError(t, err)
	require.Equal(t, 0, exitCode)

	for _, file := range out.Files {
		fileName := filepath.Base(file.Path)

		// Read generated file from disk
		b, err := ioutil.ReadFile(filepath.Join(tmp, in.Generate, fileName))
		require.NoError(t, err)
		generated := stripGenerated(string(b))

		// Compare with testdata
		b, err = ioutil.ReadFile(filepath.Join("testdata", fileName))
		require.NoError(t, err)
		ref := string(b)
		ref = stripGenerated(ref)
		require.Equal(t, ref, generated,
			fmt.Sprintf(
				"generated should match pkg/cmdconfig/testdata/%s", fileName))
	}
}
