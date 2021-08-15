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
	_, err := GetPath(appDir, "")
	require.Error(t, err, "assumed path does not exist ", appDir)

	appDir = "/"
	env := "foo"
	p, err := GetPath(appDir, env)
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
	prefix := "APP_"
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

	env := "dev"
	prefix := "APP_"
	appDir := os.Getenv("APP_DIR")
	require.NotEmpty(t, appDir)

	in := &CmdIn{}
	in.AppDir = appDir
	in.DryRun = new(bool)
	*in.DryRun = true
	in.Prefix = &prefix
	in.Env = &env
	in.Compare = new(string)
	// Path to generate config helper,
	// dry run is set so existing file won't be overwritten
	generate := filepath.Join("pkg", "cmdconfig", "testdata")
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

	files := strings.Split(generated, "// FilePath: ")
	verified := 0
	for _, file := range files {
		index := strings.Index(file, "\n")
		if index > 0 {
			filePath := file[:index]
			fileName := filepath.Base(filePath)
			generated := file[index:]
			generated = StripGenerated(generated)
			// Test fixtures in Go
			// https://dave.cheney.net/2016/05/10/test-fixtures-in-go
			b, err := ioutil.ReadFile(filepath.Join("testdata", fileName))
			require.NoError(t, err)
			ref := string(b)
			ref = StripGenerated(ref)
			require.Equal(t, ref, generated,
				fmt.Sprintf(
					"generated should match pkg/cmdconfig/testdata/%s", fileName))
			verified++
		}
	}
	require.Equal(t, 2, verified,
		"Unexpected number of files verified")

	// We've checked the generated code match the files in pkg/cmdconfig/testdata,
	// now check the generated code works as expected...
	err = os.Setenv("APP_DIR", filepath.Join(appDir, generate))
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
	prefix := "APP_"
	in.Prefix = &prefix
	in.Env = &env
	keys := ArgMap{"APP_FOO", "APP_bar"}
	values := ArgMap{"update 1", "update 2"}
	in.Keys = &keys
	in.Values = &values
	in.Compare = new(string)
	in.Generate = new(string)
	in.Config, err = NewConfig(in.AppDir, *in.Env, *in.Prefix)
	flagBase64 := false
	in.Base64 = &flagBase64
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
	prefix := "APP_"
	in.Prefix = &prefix
	in.Env = &env
	in.Config, err = NewConfig(in.AppDir, *in.Env, *in.Prefix)
	require.NoError(t, err)
	csv := false
	in.CSV = &csv

	buf, err := SetEnv(in)
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
	prefix := "APP_"
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

	require.Equal(t, "APP_BAR=bar,APP_FOO=foo", out.Buf.String())
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
	prefix := "APP_"
	in.Prefix = &prefix
	in.Env = &env
	compare := ""
	in.Compare = &compare
	generate := ""
	in.Generate = &generate
	flagBase64 := true
	in.Base64 = &flagBase64
	in.Config, err = NewConfig(in.AppDir, *in.Env, *in.Prefix)
	csv := false
	in.CSV = &csv
	require.NoError(t, err)

	out, err := Cmd(in)
	require.NoError(t, err)
	require.Equal(t, "base64", out.Cmd)
	require.Equal(t, 0, out.ExitCode)

	actual := out.Buf.String()
	require.Equal(t, "eyJBUFBfQkFSIjoiYmFyIiwiQVBQX0ZPTyI6ImZvbyJ9", actual)

	decoded, err := base64.StdEncoding.DecodeString(actual)
	require.NoError(t, err)
	require.Equal(t, `{"APP_BAR":"bar","APP_FOO":"foo"}`, string(decoded))
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
