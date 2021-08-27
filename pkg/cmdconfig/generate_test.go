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

func TestGenerateHelper(t *testing.T) {
	var err error

	appDir := os.Getenv("APP_DIR")
	require.NotEmpty(t, appDir, "APP_DIR must not be empty")

	in := &CmdIn{}
	in.AppDir = appDir
	in.DryRun = true
	in.Prefix = "APP_"
	in.Env = "dev"
	// Path to generate config helper,
	// dry run is set so existing file won't be overwritten
	in.Generate = filepath.Join("pkg", "cmdconfig", "testdata")
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
		// Test fixtures in Go
		// https://dave.cheney.net/2016/05/10/test-fixtures-in-go
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
