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
	_, err := getConfigFilePath(appDir, "", FileTypeJSON)
	require.Error(t, err, "assumed path does not exist ", appDir)

	appDir = "/"
	env := "foo"
	p, err := getConfigFilePath(appDir, env, FileTypeJSON)
	require.NoError(t, err)
	require.Equal(t, filepath.Join(appDir, fmt.Sprintf("config.%v.json", env)), p)
}

func TestFileTypes(t *testing.T) {
	configPaths := []string{
		".env",
		"prod.env",
		"sample.dev.env",
	}
	for _, configPath := range configPaths {
		fileType := filepath.Ext(configPath)
		require.Equal(t, FileTypeEnv, fileType)
	}
	configPaths = []string{
		"config.json",
		"config.prod.json",
		"sample.config.dev.json",
	}
	for _, configPath := range configPaths {
		fileType := filepath.Ext(configPath)
		require.Equal(t, FileTypeJSON, fileType)
	}
	configPaths = []string{
		"config.yaml",
		"config.prod.yaml",
		"sample.config.dev.yaml",
	}
	for _, configPath := range configPaths {
		fileType := filepath.Ext(configPath)
		require.Equal(t, FileTypeYAML, fileType)
	}
}

func TestNewConfigENV(t *testing.T) {
	tmp, err := ioutil.TempDir("", "mozey-config")
	require.NoError(t, err)
	defer (func() {
		_ = os.RemoveAll(tmp)
	})()

	env := "dev"

	configPath := filepath.Join(tmp, ".env")
	err = ioutil.WriteFile(
		configPath,
		[]byte("APP_FOO=foo\nAPP_BAR=bar\n"),
		0644)
	require.NoError(t, err)

	_, config, err := newConf(tmp, env)
	require.NoError(t, err)
	require.Len(t, config.Keys, 2)
	require.Equal(t, config.Map["APP_FOO"], "foo")
	require.Equal(t, config.Map["APP_BAR"], "bar")

	err = os.Remove(configPath)
	require.NoError(t, err)
}

func TestNewConfigJSON(t *testing.T) {
	tmp, err := ioutil.TempDir("", "mozey-config")
	require.NoError(t, err)
	defer (func() {
		_ = os.RemoveAll(tmp)
	})()

	env := "dev"

	configPath := filepath.Join(tmp, "config.json")
	err = ioutil.WriteFile(
		configPath,
		[]byte(`{"APP_FOO": "foo", "APP_BAR": "bar"}`),
		0644)
	require.NoError(t, err)

	_, config, err := newConf(tmp, env)
	require.NoError(t, err)
	require.Len(t, config.Keys, 2)
	require.Equal(t, config.Map["APP_FOO"], "foo")
	require.Equal(t, config.Map["APP_BAR"], "bar")

	err = os.Remove(configPath)
	require.NoError(t, err)
}

func TestNewConfigYAML(t *testing.T) {
	tmp, err := ioutil.TempDir("", "mozey-config")
	require.NoError(t, err)
	defer (func() {
		_ = os.RemoveAll(tmp)
	})()

	env := "dev"

	configPath := filepath.Join(tmp, "config.yaml")
	err = ioutil.WriteFile(
		configPath,
		[]byte("APP_FOO: foo\nAPP_BAR: bar\n"),
		0644)
	require.NoError(t, err)

	_, config, err := newConf(tmp, env)
	require.NoError(t, err)
	require.Len(t, config.Keys, 2)
	require.Equal(t, config.Map["APP_FOO"], "foo")
	require.Equal(t, config.Map["APP_BAR"], "bar")

	err = os.Remove(configPath)
	require.NoError(t, err)
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

	out, err := Cmd(in)
	require.NoError(t, err)
	require.Equal(t, CmdCompare, out.Cmd)
	require.Equal(t,
		"APP_BAR\nAPP_FOO\n",
		out.Buf.String(), "Non-matching keys must be listed")
	require.Equal(t, 1, out.ExitCode)
}

func TestUpdateConfigSingleJSON(t *testing.T) {
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

	out, err := Cmd(in)
	require.NoError(t, err)
	require.Equal(t, CmdUpdateConfig, out.Cmd)
	require.Equal(t, 0, out.ExitCode)

	m := make(map[string]string)
	err = json.Unmarshal(out.Files[0].Buf.Bytes(), &m)
	require.NoError(t, err)
	require.Empty(t, m["APP_DIR"], "APP_DIR must not be set in config file")
	require.Equal(t, "update 1", m["APP_FOO"])
	// 2021-08-15 Use keys exactly as per config file
	// require.Empty(t, m["APP_bar"], "keys must be uppercase")
	require.Equal(t, "update 2", m["APP_bar"])
}

func TestUpdateConfigMulti(t *testing.T) {
	tmp, err := ioutil.TempDir("", "mozey-config")
	require.NoError(t, err)
	defer (func() {
		// log.Info().Str("tmp", tmp).Msg("")
		_ = os.RemoveAll(tmp)
	})()

	// Create config files...
	test0 := "xxx"

	env := "dev"
	// Non-sample
	err = ioutil.WriteFile(
		filepath.Join(tmp, fmt.Sprintf("config.%v.json", env)),
		[]byte(fmt.Sprintf(`{"APP_FOO": "%s"}`, test0)),
		0644)
	require.NoError(t, err)
	// Sample
	err = ioutil.WriteFile(
		filepath.Join(tmp, fmt.Sprintf("sample.config.%v.json", env)),
		[]byte(fmt.Sprintf(`{"APP_FOO": "%s"}`, test0)),
		0644)
	require.NoError(t, err)

	env = "prod"
	// Non-sample
	err = ioutil.WriteFile(
		filepath.Join(tmp, fmt.Sprintf("config.%v.json", env)),
		[]byte(fmt.Sprintf(`{"APP_FOO": "%s"}`, test0)),
		0644)
	require.NoError(t, err)
	// Sample
	err = ioutil.WriteFile(
		filepath.Join(tmp, fmt.Sprintf("sample.config.%v.json", env)),
		[]byte(fmt.Sprintf(`{"APP_FOO": "%s"}`, test0)),
		0644)
	require.NoError(t, err)

	var in *CmdIn
	var out *CmdOut

	// .........................................................................
	test1 := "Only the file as the env flag"
	in = &CmdIn{}
	in.AppDir = tmp
	in.Prefix = "APP_"
	in.Env = "dev"
	in.Keys = ArgMap{"APP_FOO"}
	in.Values = ArgMap{test1}
	out, err = Cmd(in)
	require.NoError(t, err)
	require.Equal(t, CmdUpdateConfig, out.Cmd)
	require.Equal(t, 0, out.ExitCode)
	require.Len(t, out.Files, 1)
	file := out.Files[0]
	require.Contains(t, file.Path, "config.dev.json")
	m := make(map[string]string)
	err = json.Unmarshal(file.Buf.Bytes(), &m)
	require.NoError(t, err)
	require.Equal(t, test1, m["APP_FOO"], file.Path)

	// .........................................................................
	test2 := "Only the non-sample files"
	in = &CmdIn{}
	in.AppDir = tmp
	in.Prefix = "APP_"
	in.Env = "*"
	in.Keys = ArgMap{"APP_FOO"}
	in.Values = ArgMap{test2}
	out, err = Cmd(in)
	require.NoError(t, err)
	require.Equal(t, CmdUpdateConfig, out.Cmd)
	require.Equal(t, 0, out.ExitCode)
	for _, file := range out.Files {
		m := make(map[string]string)
		err = json.Unmarshal(file.Buf.Bytes(), &m)
		require.NoError(t, err)
		if strings.Contains(file.Path, "config.dev.json") ||
			strings.Contains(file.Path, "config.prod.json") {
			require.Equal(t, test2, m["APP_FOO"], file.Path)
		} else {
			t.Errorf("unexpected path %s", file.Path)
		}
	}

	// .........................................................................
	test3 := "Only the sample files"
	in = &CmdIn{}
	in.AppDir = tmp
	in.Prefix = "APP_"
	in.Env = "sample.*"
	in.Keys = ArgMap{"APP_FOO"}
	in.Values = ArgMap{test3}
	out, err = Cmd(in)
	require.NoError(t, err)
	require.Equal(t, CmdUpdateConfig, out.Cmd)
	require.Equal(t, 0, out.ExitCode)
	for _, file := range out.Files {
		m := make(map[string]string)
		err = json.Unmarshal(file.Buf.Bytes(), &m)
		require.NoError(t, err)
		if strings.Contains(file.Path, "sample.config.dev.json") ||
			strings.Contains(file.Path, "sample.config.prod.json") {
			require.Equal(t, test3, m["APP_FOO"], file.Path)
		} else {
			t.Errorf("unexpected path %s", file.Path)
		}
	}

	// .........................................................................
	test4 := "All the files"
	in = &CmdIn{}
	in.AppDir = tmp
	in.Prefix = "APP_"
	in.All = true
	in.Keys = ArgMap{"APP_FOO"}
	in.Values = ArgMap{test4}
	out, err = Cmd(in)
	require.NoError(t, err)
	require.Equal(t, CmdUpdateConfig, out.Cmd)
	require.Equal(t, 0, out.ExitCode)
	log.Debug().
		Interface("test", test4).
		Int("len", len(out.Files)).
		Msg("")
	for _, file := range out.Files {
		m := make(map[string]string)
		err = json.Unmarshal(file.Buf.Bytes(), &m)
		require.NoError(t, err)
		require.Equal(t, test4, m["APP_FOO"], file.Path)
	}

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

func TestGetEnvs(t *testing.T) {
	tmp, err := ioutil.TempDir("", "mozey-config")
	require.NoError(t, err)
	defer (func() {
		_ = os.RemoveAll(tmp)
	})()

	err = ioutil.WriteFile(
		filepath.Join(tmp, "config.dev.json"),
		[]byte(`{}`),
		0644)
	require.NoError(t, err)
	err = ioutil.WriteFile(
		filepath.Join(tmp, "sample.config.dev.json"),
		[]byte(`{}`),
		0644)
	require.NoError(t, err)
	err = ioutil.WriteFile(
		filepath.Join(tmp, "config.prod.json"),
		[]byte(`{}`),
		0644)
	require.NoError(t, err)
	err = ioutil.WriteFile(
		filepath.Join(tmp, "sample.config.prod.json"),
		[]byte(`{}`),
		0644)
	require.NoError(t, err)

	envs, err := getEnvs(tmp, false)
	require.NoError(t, err)
	require.Equal(t, []string{"dev", "prod"}, envs)

	envs, err = getEnvs(tmp, true)
	require.NoError(t, err)
	require.Equal(t, []string{"sample.dev", "sample.prod"}, envs)
}
