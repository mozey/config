package config

import (
	"encoding/json"
	"fmt"
	config "github.com/mozey/config/pkg/config/testdata"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/require"
	"io/ioutil"
	"math/rand"
	"os"
	"path"
	"strings"
	"testing"
	"time"
)

func init() {
}

// https://stackoverflow.com/a/22892986/639133
var letters = []rune("ABCDEFGHIJKLMNOPQRSTUVWXYZ")

func randString(n int) string {
	rand.Seed(time.Now().UnixNano())
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

func TestGetPath(t *testing.T) {
	appDir := randString(8)
	_, err := GetPath(appDir, "")
	require.Error(t, err, "assumed path does not exist ", appDir)

	appDir = "/"
	env := "foo"
	p, err := GetPath(appDir, env)
	require.NoError(t, err)
	require.Equal(t, path.Join(appDir, fmt.Sprintf("config.%v.json", env)), p)
}

func TestCompareKeys(t *testing.T) {
	tmp, err := ioutil.TempDir("", "mozey-config")
	require.NoError(t, err)
	defer (func() {
		_ = os.RemoveAll(tmp)
	})()

	env := "dev"
	compare := "prod"

	err = ioutil.WriteFile(
		path.Join(tmp, fmt.Sprintf("config.%v.json", env)),
		[]byte(`{"APP_ONE": "1", "APP_FOO": "foo"}`),
		0644)
	require.NoError(t, err)
	err = ioutil.WriteFile(
		path.Join(tmp, fmt.Sprintf("config.%v.json", compare)),
		[]byte(`{"APP_BAR": "bar", "APP_ONE": "1"}`),
		0644)
	require.NoError(t, err)

	in := &CmdIn{}
	in.AppDir = tmp
	prefix := "APP"
	in.Prefix = &prefix
	in.Env = &env
	in.Compare = &compare
	in.Config, err = NewConfig(in.AppDir, *in.Env, *in.Prefix)
	require.NoError(t, err)
	csv := false
	in.CSV = &csv

	out, err := Cmd(in)
	require.NoError(t, err)
	require.Equal(t, "compare", out.Cmd)
	require.Equal(t, 1, out.ExitCode)
	require.Equal(t, "APP_BAR\nAPP_FOO\n", out.Buf.String())
}

func TestGenerateHelper(t *testing.T) {
	var err error

	env := "dev"
	prefix := "APP"
	appDir := os.Getenv("APP_DIR")
	require.NotEmpty(t, appDir)

	// Path to generate config helper,
	// existing file won't be overwritten
	generate := path.Join(appDir, "pkg", "config", "testdata")

	in := &CmdIn{}
	in.AppDir = os.Getenv("APP_DIR")
	in.Prefix = &prefix
	in.Env = &env
	in.Compare = new(string)
	in.Generate = &generate
	in.Config, err = NewConfig("testdata", *in.Env, *in.Prefix)
	require.NoError(t, err)
	csv := false
	in.CSV = &csv

	out, err := Cmd(in)
	require.NoError(t, err)
	require.Equal(t, "generate", out.Cmd)
	require.Equal(t, 0, out.ExitCode)
	generated := out.Buf.String()
	//log.Debug().Msg(generated)
	log.Debug().Msg("generated should match cmd/config/testdata/config.go")

	// Validate generated code
	// https://dave.cheney.net/2016/05/10/test-fixtures-in-go
	b, err := ioutil.ReadFile("testdata/config.go")
	generated = strings.Replace(generated, " ", "", -1)
	generated = strings.Replace(generated, "\t", "", -1)
	generated = strings.Replace(generated, "\n", "", -1)
	ref := string(b)
	ref = strings.Replace(ref, " ", "", -1)
	ref = strings.Replace(ref, "\t", "", -1)
	ref = strings.Replace(ref, "\n", "", -1)
	require.Equal(t, ref, generated)

	// Use config.dev.json from testdata
	err = os.Setenv("APP_DIR", generate)
	require.NoError(t, err)
	c, err := config.LoadFile("dev")
	require.NoError(t, err)
	err = os.Setenv("APP_DIR", appDir)
	require.NoError(t, err)
	require.Equal(t, "foo", c.Foo())
	require.Equal(t, "bar", c.Bar())
}

func TestUpdateConfig(t *testing.T) {
	tmp, err := ioutil.TempDir("", "mozey-config")
	require.NoError(t, err)
	defer (func() {
		_ = os.RemoveAll(tmp)
	})()

	env := "dev"

	err = ioutil.WriteFile(
		path.Join(tmp, fmt.Sprintf("config.%v.json", env)),
		[]byte(`{"APP_FOO": "foo", "APP_BAR": "bar"}`),
		0644)

	in := &CmdIn{}
	in.AppDir = tmp
	prefix := "APP"
	in.Prefix = &prefix
	in.Env = &env
	keys := ArgMap{"APP_FOO", "APP_BAR"}
	values := ArgMap{"update 1", "update 2"}
	in.Keys = &keys
	in.Values = &values
	in.Compare = new(string)
	in.Generate = new(string)
	in.Config, err = NewConfig(in.AppDir, *in.Env, *in.Prefix)
	require.NoError(t, err)
	csv := false
	in.CSV = &csv

	out, err := Cmd(in)
	require.NoError(t, err)
	require.Equal(t, "update_config", out.Cmd)
	require.Equal(t, 0, out.ExitCode)
	log.Debug().Msg(out.Buf.String())

	m := make(map[string]string)
	err = json.Unmarshal(out.Buf.Bytes(), &m)
	require.NoError(t, err)
	require.Equal(t, "update 1", m["APP_FOO"])
	require.Equal(t, "update 2", m["APP_BAR"])
}

func TestSetEnv(t *testing.T) {
	tmp, err := ioutil.TempDir("", "mozey-config")
	require.NoError(t, err)
	defer (func() {
		_ = os.RemoveAll(tmp)
	})()

	env := "dev"

	err = ioutil.WriteFile(
		path.Join(tmp, fmt.Sprintf("config.%v.json", env)),
		[]byte(`{"APP_BAR": "bar"}`),
		0644)

	err = os.Setenv("APP_FOO", "foo")
	require.NoError(t, err)

	in := &CmdIn{}
	in.AppDir = tmp
	prefix := "APP"
	in.Prefix = &prefix
	in.Env = &env
	in.Config, err = NewConfig(in.AppDir, *in.Env, *in.Prefix)
	require.NoError(t, err)
	csv := false
	in.CSV = &csv

	buf, err := SetEnv(in)
	require.NoError(t, err)
	s := buf.String()
	require.Contains(t, s, "export APP_BAR=bar\n")
	require.Contains(t, s, "export APP_DIR=")
	require.Contains(t, s, "unset APP_FOO\n")
}

func TestCSV(t *testing.T) {
	tmp, err := ioutil.TempDir("", "mozey-config")
	require.NoError(t, err)
	defer (func() {
		_ = os.RemoveAll(tmp)
	})()

	env := "dev"

	err = ioutil.WriteFile(
		path.Join(tmp, fmt.Sprintf("config.%v.json", env)),
		[]byte(`{"APP_FOO": "foo", "APP_BAR": "bar"}`),
		0644)

	in := &CmdIn{}
	in.AppDir = tmp
	prefix := "APP"
	in.Prefix = &prefix
	in.Env = &env
	in.Compare = new(string)
	in.Generate = new(string)
	in.Config, err = NewConfig(in.AppDir, *in.Env, *in.Prefix)
	csv := true
	in.CSV = &csv
	require.NoError(t, err)

	out, err := Cmd(in)
	require.NoError(t, err)
	require.Equal(t, "csv", out.Cmd)
	require.Equal(t, 0, out.ExitCode)

	e := fmt.Sprintf("APP_BAR=bar,APP_DIR=%v,APP_FOO=foo", in.AppDir)
	require.Equal(t, e, out.Buf.String())
}
