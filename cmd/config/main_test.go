package main

import (
	"encoding/json"
	"fmt"
	"github.com/mozey/logutil"
	"github.com/stretchr/testify/require"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"path"
	"testing"
	"time"
)

func init() {
	// Setup logging
	log.SetFlags(log.Ldate | log.Ltime | log.LUTC | log.Lshortfile)
	logutil.SetDebug(true)
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
	defer os.RemoveAll(tmp)

	env := "dev"
	compare := "prod"

	ioutil.WriteFile(
		path.Join(tmp, fmt.Sprintf("config.%v.json", env)),
		[]byte(`{"APP_ONE": "1", "APP_FOO": "foo"}`),
		0644)
	ioutil.WriteFile(
		path.Join(tmp, fmt.Sprintf("config.%v.json", compare)),
		[]byte(`{"APP_BAR": "bar", "APP_ONE": "1"}`),
		0644)

	in := &CmdIn{}
	in.AppDir = tmp
	prefix := "APP"
	in.Prefix = &prefix
	in.Env = &env
	in.Compare = &compare
	in.Config, err = NewConfig(in.AppDir, *in.Env, *in.Prefix)
	require.NoError(t, err)

	out, err := Cmd(in)
	require.NoError(t, err)
	require.Equal(t, "compare", out.Cmd)
	require.Equal(t, 1, out.ExitCode)
	require.Equal(t, "APP_BAR\nAPP_FOO\n", out.Buf.String())
}

func TestGenerateHelper(t *testing.T) {
	tmp, err := ioutil.TempDir("", "mozey-config")
	require.NoError(t, err)
	defer os.RemoveAll(tmp)

	env := "dev"
	prefix := "APP"

	appDir := path.Join(tmp, "src", "app")
	os.MkdirAll(appDir, os.ModePerm)
	err = ioutil.WriteFile(
		path.Join(appDir, fmt.Sprintf("config.%v.json", env)),
		[]byte(`{"APP_FOO": "foo", "APP_BAR": "bar"}`),
		0644)
	require.NoError(t, err)

	in := &CmdIn{}
	in.AppDir = appDir
	in.Prefix = &prefix
	in.Env = &env
	in.Compare = new(string)
	generate := "src/app"
	in.Generate = &generate
	in.Config, err = NewConfig(in.AppDir, *in.Env, *in.Prefix)
	require.NoError(t, err)

	out, err := Cmd(in)
	require.NoError(t, err)
	require.Equal(t, "generate", out.Cmd)
	require.Equal(t, 0, out.ExitCode)
	logutil.Debug(out.Buf.String())
	// TODO Validate generated code
}

func TestUpdateConfig(t *testing.T) {
	tmp, err := ioutil.TempDir("", "mozey-config")
	require.NoError(t, err)
	defer os.RemoveAll(tmp)

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

	out, err := Cmd(in)
	require.NoError(t, err)
	require.Equal(t, "update_config", out.Cmd)
	require.Equal(t, 0, out.ExitCode)
	logutil.Debug(out.Buf.String())

	m := make(map[string]string)
	err = json.Unmarshal(out.Buf.Bytes(), &m)
	require.NoError(t, err)
	require.Equal(t, "update 1", m["APP_FOO"])
	require.Equal(t, "update 2", m["APP_BAR"])
}

func TestSetEnv(t *testing.T) {
	tmp, err := ioutil.TempDir("", "mozey-config")
	require.NoError(t, err)
	defer os.RemoveAll(tmp)

	env := "dev"

	err = ioutil.WriteFile(
		path.Join(tmp, fmt.Sprintf("config.%v.json", env)),
		[]byte(`{"APP_BAR": "bar"}`),
		0644)

	os.Setenv("APP_FOO", "foo")

	in := &CmdIn{}
	in.AppDir = tmp
	prefix := "APP"
	in.Prefix = &prefix
	in.Env = &env
	in.Config, err = NewConfig(in.AppDir, *in.Env, *in.Prefix)
	require.NoError(t, err)

	buf, err := SetEnv(in)
	require.NoError(t, err)
	s := buf.String()
	require.Contains(t, s, "export APP_BAR=bar\n")
	require.Contains(t, s, "export APP_DIR=")
	require.Contains(t, s, "unset APP_FOO\n")
}