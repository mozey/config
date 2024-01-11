package cmdconfig

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	config "github.com/mozey/config/pkg/cmdconfig/testdata"
	"github.com/mozey/config/pkg/testutil"
	"github.com/pkg/errors"
)

func stripGenerated(generated string) string {
	generated = strings.Replace(generated, " ", "", -1)
	generated = strings.Replace(generated, "\t", "", -1)
	generated = strings.Replace(generated, "\n", "", -1)
	return generated
}

func TestGenerateHelpersPrint(t *testing.T) {
	is := testutil.Setup(t)
	var err error

	appDir := os.Getenv("APP_DIR")
	is.True(appDir != "") // APP_DIR must not be empty

	in := &CmdIn{}
	in.DryRun = true // Do not write files to disk
	in.Prefix = "APP_"
	in.Env = "dev"

	// Path to generate config helpers is not used since dry run is set.
	// Compare with TestGenerateHelpers
	in.Generate = filepath.Join("pkg", "cmdconfig", "testdata")

	in.AppDir = filepath.Join(appDir, in.Generate)

	out, err := Cmd(in)
	is.NoErr(err)
	is.Equal(CmdGenerate, out.Cmd)
	is.Equal(0, out.ExitCode)
	is.Equal(3, len(out.Files)) // Unexpected number of files

	is.Equal(len(out.Files), 3) // Count generated file
	for _, file := range out.Files {
		is.True(strings.TrimSpace(file.Path) != "") // File path empty
		fileName := filepath.Base(file.Path)
		os.WriteFile(
			filepath.Join("testdata", "compare", fileName),
			file.Buf.Bytes(), 0644)
		generated := stripGenerated(file.Buf.String())
		// See "Test fixtures in Go"
		// https://dave.cheney.net/2016/05/10/test-fixtures-in-go
		b, err := os.ReadFile(filepath.Join("testdata", fileName))
		is.NoErr(err)
		ref := string(b)
		ref = stripGenerated(ref)
		if ref != generated {
			is.NoErr(errors.Errorf(
				"generated should match pkg/cmdconfig/testdata/%s", fileName))
		}
	}

	// We've checked the generated code match the files in pkg/cmdconfig/testdata,
	// now check the generated code works as expected...
	err = os.Setenv("APP_DIR", filepath.Join(appDir, in.Generate))
	is.NoErr(err)
	c, err := config.LoadFile("dev")
	is.NoErr(err)
	err = os.Setenv("APP_DIR", appDir)
	is.NoErr(err)
	is.Equal("foo", c.Foo())
	is.Equal("bar", c.Bar())
	is.Equal("Buzz", c.Buz())
	is.Equal("FizzBuzz-FizzBuzz", c.ExecTemplateFiz("-FizzBuzz"))
}

// TestGenerateHelpersSave also covers Files_Save
func TestGenerateHelpersSave(t *testing.T) {
	is := testutil.Setup(t)

	tmp, err := os.MkdirTemp("", "mozey-config")
	is.NoErr(err)
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

	// Copy config file from testdata to tmp dir.
	// See "Test fixtures in Go"
	// https://dave.cheney.net/2016/05/10/test-fixtures-in-go
	configFilePath, err := getConfigFilePath("testdata", in.Env, FileTypeJSON)
	is.NoErr(err)
	dstConfigFilePath, err := getConfigFilePath(tmp, in.Env, FileTypeJSON)
	is.NoErr(err)
	err = Copy(configFilePath, dstConfigFilePath)
	is.NoErr(err)

	out, err := Cmd(in)
	is.NoErr(err)
	is.Equal(CmdGenerate, out.Cmd)
	is.Equal(0, out.ExitCode)
	is.Equal(3, len(out.Files)) // Unexpected number of files

	// Write the files
	// TODO in.Process calls fmt.Println,
	// temporarily capture stdout to avoid cluttering test output?
	// See https://github.com/mozey/go-capturer
	exitCode, err := in.Process(out)
	is.NoErr(err)
	is.Equal(0, exitCode)

	for _, file := range out.Files {
		fileName := filepath.Base(file.Path)

		// Read generated file from disk
		b, err := os.ReadFile(filepath.Join(tmp, in.Generate, fileName))
		is.NoErr(err)
		generated := stripGenerated(string(b))

		// Compare with testdata
		b, err = os.ReadFile(filepath.Join("testdata", fileName))
		is.NoErr(err)
		ref := string(b)
		ref = stripGenerated(ref)
		if ref != generated {
			is.NoErr(errors.Errorf(
				"generated should match pkg/cmdconfig/testdata/%s", fileName))
		}
	}
}
