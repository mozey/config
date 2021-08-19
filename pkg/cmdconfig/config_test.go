package cmdconfig

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"text/template"
	"time"

	// NOTE TestGenerateHelper checks that the code in pkg/cmdconfig/testdata
	// matches wat is actually generated. Therefore, this package can be
	// imported to test the generated code works as expected
	config "github.com/mozey/config/pkg/cmdconfig/testdata"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/require"
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
	_, err := GetConfigFilePath(appDir, "")
	require.Error(t, err, "assumed path does not exist ", appDir)

	appDir = "/"
	env := "foo"
	p, err := GetConfigFilePath(appDir, env)
	require.NoError(t, err)
	require.Equal(t, filepath.Join(appDir, fmt.Sprintf("config.%v.json", env)), p)
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
		filepath.Join(tmp, fmt.Sprintf("config.%v.json", env)),
		[]byte(`{"APP_ONE": "1", "APP_FOO": "foo"}`),
		0644)
	require.NoError(t, err)
	err = ioutil.WriteFile(
		filepath.Join(tmp, fmt.Sprintf("config.%v.json", compare)),
		[]byte(`{"APP_BAR": "bar", "APP_ONE": "1"}`),
		0644)
	require.NoError(t, err)

	in := &CmdIn{}
	in.AppDir = tmp
	in.Prefix = "APP_"
	in.Env = env
	in.Compare = compare
	in.Config, err = NewConfig(in.AppDir, in.Env, in.Prefix)
	require.NoError(t, err)

	out, err := Cmd(in)
	require.NoError(t, err)
	require.Equal(t, CmdCompare, out.Cmd)
	require.Equal(t, 1, out.ExitCode)
	require.Equal(t,
		"APP_BAR\nAPP_FOO\n",
		out.Buf.String())
}

func StripGenerated(generated string) string {
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
	in.Config, err = NewConfig("testdata", in.Env, in.Prefix)
	require.NoError(t, err)

	out, err := Cmd(in)
	require.NoError(t, err)
	require.Equal(t, CmdGenerate, out.Cmd)
	require.Equal(t, 0, out.ExitCode)
	require.Equal(t, 3, len(out.Files),
		"Unexpected number of files")

	for _, file := range out.Files {
		fileName := filepath.Base(file.Path)
		generated := StripGenerated(file.Buf.String())
		// Test fixtures in Go
		// https://dave.cheney.net/2016/05/10/test-fixtures-in-go
		b, err := ioutil.ReadFile(filepath.Join("testdata", fileName))
		require.NoError(t, err)
		ref := string(b)
		ref = StripGenerated(ref)
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

func TestUpdateConfig(t *testing.T) {
	tmp, err := ioutil.TempDir("", "mozey-config")
	require.NoError(t, err)
	defer (func() {
		_ = os.RemoveAll(tmp)
	})()

	env := "dev"

	err = ioutil.WriteFile(
		filepath.Join(tmp, fmt.Sprintf("config.%v.json", env)),
		[]byte(`{"APP_FOO": "foo", "APP_BAR": "bar"}`),
		0644)
	require.NoError(t, err)

	in := &CmdIn{}
	in.AppDir = tmp
	in.Prefix = "APP_"
	in.Env = env
	in.Keys = ArgMap{"APP_FOO", "APP_bar"}
	in.Values = ArgMap{"update 1", "update 2"}
	in.Config, err = NewConfig(in.AppDir, in.Env, in.Prefix)
	require.NoError(t, err)

	out, err := Cmd(in)
	require.NoError(t, err)
	require.Equal(t, CmdUpdateConfig, out.Cmd)
	require.Equal(t, 0, out.ExitCode)
	log.Debug().Msg(out.Buf.String())

	m := make(map[string]string)
	err = json.Unmarshal(out.Buf.Bytes(), &m)
	require.NoError(t, err)
	require.Empty(t, m["APP_DIR"], "APP_DIR must not be set in config file")
	require.Equal(t, "update 1", m["APP_FOO"])
	// 2021-08-15 Use keys exactly as per config file
	// require.Empty(t, m["APP_bar"], "keys must be uppercase")
	require.Equal(t, "update 2", m["APP_bar"])
}

func TestSetEnv(t *testing.T) {
	tmp, err := ioutil.TempDir("", "mozey-config")
	require.NoError(t, err)
	defer (func() {
		_ = os.RemoveAll(tmp)
	})()

	env := "dev"

	err = ioutil.WriteFile(
		filepath.Join(tmp, fmt.Sprintf("config.%v.json", env)),
		[]byte(`{"APP_BAR": "bar"}`),
		0644)
	require.NoError(t, err)

	err = os.Setenv("APP_FOO", "foo")
	require.NoError(t, err)

	in := &CmdIn{}
	in.AppDir = tmp
	in.Prefix = "APP_"
	in.Env = env
	in.Config, err = NewConfig(in.AppDir, in.Env, in.Prefix)
	require.NoError(t, err)

	buf, _, err := setEnv(in)
	require.NoError(t, err)
	s := buf.String()

	if runtime.GOOS == "windows" {
		require.Contains(t, s, "set APP_BAR=bar")
		require.Contains(t, s, "set APP_FOO=\"\"")
		require.NotContains(t, s, "set APP_DIR=\"\"")

	} else {
		require.Contains(t, s, "export APP_BAR=bar")
		require.Contains(t, s, "unset APP_FOO")
		require.NotContains(t, s, "unset APP_DIR")
	}
}

func TestCSV(t *testing.T) {
	tmp, err := ioutil.TempDir("", "mozey-config")
	require.NoError(t, err)
	defer (func() {
		_ = os.RemoveAll(tmp)
	})()

	env := "dev"

	err = ioutil.WriteFile(
		filepath.Join(tmp, fmt.Sprintf("config.%v.json", env)),
		[]byte(`{"APP_FOO": "foo", "APP_BAR": "bar"}`),
		0644)
	require.NoError(t, err)

	in := &CmdIn{}
	in.AppDir = tmp
	in.Prefix = "APP_"
	in.Env = env
	in.Config, err = NewConfig(in.AppDir, in.Env, in.Prefix)
	require.NoError(t, err)
	in.CSV = true

	in.Sep = ","
	out, err := Cmd(in)
	require.NoError(t, err)
	require.Equal(t, CmdCSV, out.Cmd)
	require.Equal(t, 0, out.ExitCode)
	require.Equal(t, "APP_BAR=bar,APP_FOO=foo", out.Buf.String())

	in.Sep = " "
	out, err = Cmd(in)
	require.NoError(t, err)
	require.Equal(t, CmdCSV, out.Cmd)
	require.Equal(t, 0, out.ExitCode)
	require.Equal(t, "APP_BAR=bar APP_FOO=foo", out.Buf.String())
}

func TestBase64(t *testing.T) {
	tmp, err := ioutil.TempDir("", "mozey-config")
	require.NoError(t, err)
	defer (func() {
		_ = os.RemoveAll(tmp)
	})()

	env := "dev"

	err = ioutil.WriteFile(
		filepath.Join(tmp, fmt.Sprintf("config.%v.json", env)),
		[]byte(`{"APP_FOO": "foo", "APP_BAR": "bar"}`),
		0644)
	require.NoError(t, err)

	in := &CmdIn{}
	in.AppDir = tmp
	in.Prefix = "APP_"
	in.Env = env
	in.Base64 = true
	in.Config, err = NewConfig(in.AppDir, in.Env, in.Prefix)
	require.NoError(t, err)

	out, err := Cmd(in)
	require.NoError(t, err)
	require.Equal(t, CmdBase64, out.Cmd)
	require.Equal(t, 0, out.ExitCode)

	actual := out.Buf.String()
	require.Equal(t, "eyJBUFBfQkFSIjoiYmFyIiwiQVBQX0ZPTyI6ImZvbyJ9", actual)

	decoded, err := base64.StdEncoding.DecodeString(actual)
	require.NoError(t, err)
	require.Equal(t, `{"APP_BAR":"bar","APP_FOO":"foo"}`, string(decoded))
}

func TestGet(t *testing.T) {
	tmp, err := ioutil.TempDir("", "mozey-config")
	require.NoError(t, err)
	defer (func() {
		_ = os.RemoveAll(tmp)
	})()

	env := "dev"

	err = ioutil.WriteFile(
		filepath.Join(tmp, fmt.Sprintf("config.%v.json", env)),
		[]byte(`{"APP_FOO": "foo", "APP_BAR": "bar"}`),
		0644)
	require.NoError(t, err)

	in := &CmdIn{}
	in.AppDir = tmp
	in.Prefix = "APP_"
	in.Env = env
	in.Config, err = NewConfig(in.AppDir, in.Env, in.Prefix)
	require.NoError(t, err)

	in.PrintValue = "APP_FOO"
	out, err := Cmd(in)
	require.NoError(t, err)
	require.Equal(t, CmdGet, out.Cmd)
	require.Equal(t, 0, out.ExitCode)
	actual := out.Buf.String()
	require.Equal(t, "foo", actual)

	in.PrintValue = "APP_BAR"
	out, err = Cmd(in)
	require.NoError(t, err)
	require.Equal(t, CmdGet, out.Cmd)
	require.Equal(t, 0, out.ExitCode)
	actual = out.Buf.String()
	require.Equal(t, "bar", actual)
}

func TestTypeConversionFns(t *testing.T) {
	c := config.New()

	// bool
	c.SetBar("1")
	b, err := c.FnBar().Bool()
	require.NoError(t, err)
	require.True(t, b)
	c.SetBar("true")
	b, err = c.FnBar().Bool()
	require.NoError(t, err)
	require.True(t, b)
	c.SetBar("TrUe")
	b, err = c.FnBar().Bool()
	require.NoError(t, err)
	require.True(t, b)

	c.SetBar("0")
	b, err = c.FnBar().Bool()
	require.NoError(t, err)
	require.False(t, b)
	c.SetBar("false")
	b, err = c.FnBar().Bool()
	require.NoError(t, err)
	require.False(t, b)

	c.SetBar("xxx")
	b, err = c.FnBar().Bool()
	require.Error(t, err)
	require.False(t, b)

	// float64
	c.SetBar("123.45")
	f, err := c.FnBar().Float64()
	require.NoError(t, err)
	expectedF := float64(123.45)
	require.Equal(t, expectedF, f)
	c.SetBar("xxx")
	f, err = c.FnBar().Float64()
	require.Error(t, err)
	require.Equal(t, float64(0), f)

	// int64
	c.SetBar("123")
	i, err := c.FnBar().Int64()
	require.NoError(t, err)
	expectedI := int64(123)
	require.Equal(t, expectedI, i)
	c.SetBar("xxx")
	i, err = c.FnBar().Int64()
	require.Error(t, err)
	require.Equal(t, int64(0), i)

	// string
	s := "This is a string"
	c.SetBar(s)
	require.Equal(t, s, c.FnBar().String())
}

func BenchmarkExecuteTemplate(b *testing.B) {
	templateFiz := "Fizz{{.Buz}}{{.Meh}}"
	buz := "Buzz"
	// WARNING 5055 ns/op if template is created inside loop
	t := template.Must(template.New("templateFiz").Parse(templateFiz))
	buf := bytes.Buffer{}
	for i := 0; i < b.N; i++ {
		// 539.0 ns/op
		_ = t.Execute(&buf, map[string]interface{}{
			"Buz": buz,
			"Meh": "Meh",
		})
	}
}

// BenchmarkExecuteTemplateSprintf demonstrates that using sprintf
// is much faster than using text/template.
// However, sprintf does not support named variables.
// Changing the order of variables for TEMPLATE_* keys in config.ENV.json files
// must not make ExecTemplate* methods return different values.
// Therefore going with text/template,
// avoid calling ExecTemplate* methods inside loops for now.
// TODO Investigate regex replace performance vs text/template
// https://github.com/mozey/config/issues/14
func BenchmarkExecuteTemplateSprintf(b *testing.B) {
	templateFiz := "Fizz%s%s"
	buz := "Buzz"
	for i := 0; i < b.N; i++ {
		// 80.14 ns/op
		_ = fmt.Sprintf(templateFiz, buz, "")
	}
}

func TestGetTemplateParams(t *testing.T) {
	params := GetTemplateParams("Fizz{{.Buz}}{{.Meh}}")
	require.Equal(t, []string{"Buz", "Meh"}, params)
}
