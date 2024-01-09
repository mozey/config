package cmdconfig

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
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
	"github.com/mozey/config/pkg/testutil"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
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
	is := testutil.Setup(t)

	appDir := randString(8)
	_, err := getConfigFilePath(appDir, "", FileTypeJSON)
	is.True(err != nil) // Assumed path does not exist

	appDir = "/"
	env := "foo"
	p, err := getConfigFilePath(appDir, env, FileTypeJSON)
	is.NoErr(err)
	is.Equal(filepath.Join(appDir, fmt.Sprintf("config.%v.json", env)), p)
}

func TestFileTypes(t *testing.T) {
	is := testutil.Setup(t)

	configPaths := []string{
		".env",
		"prod.env",
		"sample.dev.env",
	}
	for _, configPath := range configPaths {
		fileType := filepath.Ext(configPath)
		is.Equal(FileTypeEnv, fileType)
	}
	configPaths = []string{
		"config.json",
		"config.prod.json",
		"sample.config.dev.json",
	}
	for _, configPath := range configPaths {
		fileType := filepath.Ext(configPath)
		is.Equal(FileTypeJSON, fileType)
	}
	configPaths = []string{
		"config.yaml",
		"config.prod.yaml",
		"sample.config.dev.yaml",
	}
	for _, configPath := range configPaths {
		fileType := filepath.Ext(configPath)
		is.Equal(FileTypeYAML, fileType)
	}
}

func TestNewConfigENV(t *testing.T) {
	is := testutil.Setup(t)

	tmp, err := os.MkdirTemp("", "mozey-config")
	is.NoErr(err)
	defer (func() {
		_ = os.RemoveAll(tmp)
	})()

	env := "dev"

	configPath := filepath.Join(tmp, ".env")
	err = os.WriteFile(
		configPath,
		[]byte("APP_FOO=foo\nAPP_BAR=bar\n"),
		0644)
	is.NoErr(err)

	_, config, err := newSingleConf(tmp, env)
	is.NoErr(err)

	is.Equal(config.Map["APP_FOO"], "foo")
	is.Equal(config.Map["APP_BAR"], "bar")

	err = os.Remove(configPath)
	is.NoErr(err)
}

func TestNewConfigJSON(t *testing.T) {
	is := testutil.Setup(t)

	tmp, err := os.MkdirTemp("", "mozey-config")
	is.NoErr(err)
	defer (func() {
		_ = os.RemoveAll(tmp)
	})()

	env := "dev"

	configPath := filepath.Join(tmp, "config.json")
	err = os.WriteFile(
		configPath,
		[]byte(`{"APP_FOO": "foo", "APP_BAR": "bar"}`),
		0644)
	is.NoErr(err)

	_, config, err := newSingleConf(tmp, env)
	is.NoErr(err)
	is.Equal(len(config.Keys), 2)
	is.Equal(config.Map["APP_FOO"], "foo")
	is.Equal(config.Map["APP_BAR"], "bar")

	err = os.Remove(configPath)
	is.NoErr(err)
}

func TestNewConfigYAML(t *testing.T) {
	is := testutil.Setup(t)

	tmp, err := os.MkdirTemp("", "mozey-config")
	is.NoErr(err)
	defer (func() {
		_ = os.RemoveAll(tmp)
	})()

	env := "dev"

	configPath := filepath.Join(tmp, "config.yaml")
	err = os.WriteFile(
		configPath,
		[]byte("APP_FOO: foo\nAPP_BAR: bar\n"),
		0644)
	is.NoErr(err)

	_, config, err := newSingleConf(tmp, env)
	is.NoErr(err)
	is.Equal(len(config.Keys), 2)
	is.Equal(config.Map["APP_FOO"], "foo")
	is.Equal(config.Map["APP_BAR"], "bar")

	err = os.Remove(configPath)
	is.NoErr(err)
}

func TestCompareKeys(t *testing.T) {
	is := testutil.Setup(t)

	tmp, err := os.MkdirTemp("", "mozey-config")
	is.NoErr(err)
	defer (func() {
		_ = os.RemoveAll(tmp)
	})()

	env := "dev"
	compare := "prod"

	err = os.WriteFile(
		filepath.Join(tmp, fmt.Sprintf("config.%v.json", env)),
		[]byte(`{"APP_ONE": "1", "APP_FOO": "foo"}`),
		0644)
	is.NoErr(err)
	err = os.WriteFile(
		filepath.Join(tmp, fmt.Sprintf("config.%v.json", compare)),
		[]byte(`{"APP_BAR": "bar", "APP_ONE": "1"}`),
		0644)
	is.NoErr(err)

	in := &CmdIn{}
	in.AppDir = tmp
	in.Prefix = "APP_"
	in.Env = env
	in.Compare = compare

	out, err := Cmd(in)
	is.NoErr(err)
	is.Equal(CmdCompare, out.Cmd)
	is.Equal("APP_BAR\nAPP_FOO\n", out.Buf.String()) // Non-matching keys must be listed
	is.Equal(1, out.ExitCode)
}

func TestUpdateConfigSingleJSON(t *testing.T) {
	is := testutil.Setup(t)

	tmp, err := os.MkdirTemp("", "mozey-config")
	is.NoErr(err)
	defer (func() {
		_ = os.RemoveAll(tmp)
	})()

	env := "dev"

	err = os.WriteFile(
		filepath.Join(tmp, fmt.Sprintf("config.%v.json", env)),
		[]byte(`{"APP_FOO": "foo", "APP_BAR": "bar"}`),
		0644)
	is.NoErr(err)

	in := &CmdIn{}
	in.AppDir = tmp
	in.Prefix = "APP_"
	in.Env = env
	in.Keys = ArgMap{"APP_FOO", "APP_bar"}
	in.Values = ArgMap{"update 1", "update 2"}

	out, err := Cmd(in)
	is.NoErr(err)
	is.Equal(CmdUpdateConfig, out.Cmd)
	is.Equal(0, out.ExitCode)

	m := make(map[string]string)
	err = json.Unmarshal(out.Files[0].Buf.Bytes(), &m)
	is.NoErr(err)
	is.Equal(m["APP_DIR"], "") // APP_DIR must not be set in config file
	is.Equal("update 1", m["APP_FOO"])
	// 2021-08-15 Use keys exactly as per config file
	// Xrequire.Empty(t, m["APP_bar"], "keys must be uppercase")
	is.Equal("update 2", m["APP_bar"])
}

func TestUpdateConfigMulti(t *testing.T) {
	is := testutil.Setup(t)

	tmp, err := os.MkdirTemp("", "mozey-config")
	is.NoErr(err)
	defer (func() {
		// log.Info().Str("tmp", tmp).Msg("")
		_ = os.RemoveAll(tmp)
	})()

	// Create config files...
	test0 := "xxx"

	env := "dev"
	// Non-sample
	err = os.WriteFile(
		filepath.Join(tmp, fmt.Sprintf("config.%v.json", env)),
		[]byte(fmt.Sprintf(`{"APP_FOO": "%s"}`, test0)),
		0644)
	is.NoErr(err)
	// Sample
	err = os.WriteFile(
		filepath.Join(tmp, fmt.Sprintf("sample.config.%v.json", env)),
		[]byte(fmt.Sprintf(`{"APP_FOO": "%s"}`, test0)),
		0644)
	is.NoErr(err)

	env = "prod"
	// Non-sample
	err = os.WriteFile(
		filepath.Join(tmp, fmt.Sprintf("config.%v.json", env)),
		[]byte(fmt.Sprintf(`{"APP_FOO": "%s"}`, test0)),
		0644)
	is.NoErr(err)
	// Sample
	err = os.WriteFile(
		filepath.Join(tmp, fmt.Sprintf("sample.config.%v.json", env)),
		[]byte(fmt.Sprintf(`{"APP_FOO": "%s"}`, test0)),
		0644)
	is.NoErr(err)

	env = "stage-ec2"
	// Non-sample
	err = os.WriteFile(
		filepath.Join(tmp, fmt.Sprintf("config.%v.json", env)),
		[]byte(fmt.Sprintf(`{"APP_FOO": "%s"}`, test0)),
		0644)
	is.NoErr(err)
	// Sample
	err = os.WriteFile(
		filepath.Join(tmp, fmt.Sprintf("sample.config.%v.json", env)),
		[]byte(fmt.Sprintf(`{"APP_FOO": "%s"}`, test0)),
		0644)
	is.NoErr(err)

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
	is.NoErr(err)
	is.Equal(CmdUpdateConfig, out.Cmd)
	is.Equal(0, out.ExitCode)
	is.Equal(len(out.Files), 1)
	file := out.Files[0]
	is.True(strings.Contains(file.Path, "config.dev.json"))
	m := make(map[string]string)
	err = json.Unmarshal(file.Buf.Bytes(), &m)
	is.NoErr(err)
	if test1 != m["APP_FOO"] {
		is.NoErr(errors.Errorf("mismatch for path %s", file.Path))
	}

	// .........................................................................
	test2 := "Only the non-sample files"
	in = &CmdIn{}
	in.AppDir = tmp
	in.Prefix = "APP_"
	in.Env = "*"
	in.Keys = ArgMap{"APP_FOO"}
	in.Values = ArgMap{test2}
	out, err = Cmd(in)
	is.NoErr(err)
	is.Equal(CmdUpdateConfig, out.Cmd)
	is.Equal(0, out.ExitCode)
	is.Equal(len(out.Files), 3)
	for _, file := range out.Files {
		m := make(map[string]string)
		err = json.Unmarshal(file.Buf.Bytes(), &m)
		is.NoErr(err)
		if strings.Contains(file.Path, "config.dev.json") ||
			strings.Contains(file.Path, "config.stage-ec2.json") ||
			strings.Contains(file.Path, "config.prod.json") {
			if test2 != m["APP_FOO"] {
				is.NoErr(errors.Errorf("mismatch for path %s", file.Path))
			}
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
	is.NoErr(err)
	is.Equal(CmdUpdateConfig, out.Cmd)
	is.Equal(0, out.ExitCode)
	is.Equal(len(out.Files), 3)
	for _, file := range out.Files {
		m := make(map[string]string)
		err = json.Unmarshal(file.Buf.Bytes(), &m)
		is.NoErr(err)
		if strings.Contains(file.Path, "sample.config.dev.json") ||
			strings.Contains(file.Path, "sample.config.stage-ec2.json") ||
			strings.Contains(file.Path, "sample.config.prod.json") {
			if test3 != m["APP_FOO"] {
				is.NoErr(errors.Errorf("mismatch for path %s", file.Path))
			}
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
	is.NoErr(err)
	is.Equal(CmdUpdateConfig, out.Cmd)
	is.Equal(0, out.ExitCode)
	log.Debug().
		Interface("test", test4).
		Int("len", len(out.Files)).
		Msg("")
	is.Equal(len(out.Files), 6)
	for _, file := range out.Files {
		m := make(map[string]string)
		err = json.Unmarshal(file.Buf.Bytes(), &m)
		is.NoErr(err)
		if test4 != m["APP_FOO"] {
			is.NoErr(errors.Errorf("mismatch for path %s", file.Path))
		}
	}

}

func TestSetEnv(t *testing.T) {
	is := testutil.Setup(t)

	tmp, err := os.MkdirTemp("", "mozey-config")
	is.NoErr(err)
	defer (func() {
		_ = os.RemoveAll(tmp)
	})()

	env := "dev"

	err = os.WriteFile(
		filepath.Join(tmp, fmt.Sprintf("config.%v.json", env)),
		[]byte(`{"APP_BAR": "bar"}`),
		0644)
	is.NoErr(err)

	err = os.Setenv("APP_FOO", "foo")
	is.NoErr(err)

	in := &CmdIn{}
	in.AppDir = tmp
	in.Prefix = "APP_"
	in.Env = env

	buf, _, err := setEnv(in)
	is.NoErr(err)
	s := buf.String()

	if runtime.GOOS == "windows" {
		is.True(strings.Contains(s, "set APP_BAR=bar"))
		is.True(strings.Contains(s, "set APP_FOO=\"\""))
		is.True(!strings.Contains(s, "set APP_DIR=\"\""))

	} else {
		is.True(strings.Contains(s, "export APP_BAR=bar"))
		is.True(strings.Contains(s, "unset APP_FOO"))
		is.True(!strings.Contains(s, "unset APP_DIR"))
	}
}

func TestCSV(t *testing.T) {
	is := testutil.Setup(t)

	tmp, err := os.MkdirTemp("", "mozey-config")
	is.NoErr(err)
	defer (func() {
		_ = os.RemoveAll(tmp)
	})()

	env := "dev"

	err = os.WriteFile(
		filepath.Join(tmp, fmt.Sprintf("config.%v.json", env)),
		[]byte(`{"APP_FOO": "foo", "APP_BAR": "bar"}`),
		0644)
	is.NoErr(err)

	in := &CmdIn{}
	in.AppDir = tmp
	in.Prefix = "APP_"
	in.Env = env
	in.CSV = true

	in.Sep = ","
	out, err := Cmd(in)
	is.NoErr(err)
	is.Equal(CmdCSV, out.Cmd)
	is.Equal(0, out.ExitCode)
	is.Equal("APP_BAR=bar,APP_FOO=foo", out.Buf.String())

	in.Sep = " "
	out, err = Cmd(in)
	is.NoErr(err)
	is.Equal(CmdCSV, out.Cmd)
	is.Equal(0, out.ExitCode)
	is.Equal("APP_BAR=bar APP_FOO=foo", out.Buf.String())
}

func TestBase64(t *testing.T) {
	is := testutil.Setup(t)

	tmp, err := os.MkdirTemp("", "mozey-config")
	is.NoErr(err)
	defer (func() {
		_ = os.RemoveAll(tmp)
	})()

	env := "dev"

	err = os.WriteFile(
		filepath.Join(tmp, fmt.Sprintf("config.%v.json", env)),
		[]byte(`{"APP_FOO": "foo", "APP_BAR": "bar"}`),
		0644)
	is.NoErr(err)

	in := &CmdIn{}
	in.AppDir = tmp
	in.Prefix = "APP_"
	in.Env = env
	in.Base64 = true

	out, err := Cmd(in)
	is.NoErr(err)
	is.Equal(CmdBase64, out.Cmd)
	is.Equal(0, out.ExitCode)

	actual := out.Buf.String()
	is.Equal("eyJBUFBfQkFSIjoiYmFyIiwiQVBQX0ZPTyI6ImZvbyJ9", actual)

	decoded, err := base64.StdEncoding.DecodeString(actual)
	is.NoErr(err)
	is.Equal(`{"APP_BAR":"bar","APP_FOO":"foo"}`, string(decoded))
}

func TestGet(t *testing.T) {
	is := testutil.Setup(t)

	tmp, err := os.MkdirTemp("", "mozey-config")
	is.NoErr(err)
	defer (func() {
		_ = os.RemoveAll(tmp)
	})()

	env := "dev"

	err = os.WriteFile(
		filepath.Join(tmp, fmt.Sprintf("config.%v.json", env)),
		[]byte(`{"APP_FOO": "foo", "APP_BAR": "bar"}`),
		0644)
	is.NoErr(err)

	in := &CmdIn{}
	in.AppDir = tmp
	in.Prefix = "APP_"
	in.Env = env

	in.PrintValue = "APP_FOO"
	out, err := Cmd(in)
	is.NoErr(err)
	is.Equal(CmdGet, out.Cmd)
	is.Equal(0, out.ExitCode)
	actual := out.Buf.String()
	is.Equal("foo", actual)

	in.PrintValue = "APP_BAR"
	out, err = Cmd(in)
	is.NoErr(err)
	is.Equal(CmdGet, out.Cmd)
	is.Equal(0, out.ExitCode)
	actual = out.Buf.String()
	is.Equal("bar", actual)
}

func TestTypeConversionFns(t *testing.T) {
	is := testutil.Setup(t)

	c := config.New()

	// bool
	c.SetBar("1")
	b, err := c.FnBar().Bool()
	is.NoErr(err)
	is.True(b)
	c.SetBar("true")
	b, err = c.FnBar().Bool()
	is.NoErr(err)
	is.True(b)
	c.SetBar("TrUe")
	b, err = c.FnBar().Bool()
	is.NoErr(err)
	is.True(b)

	c.SetBar("0")
	b, err = c.FnBar().Bool()
	is.NoErr(err)
	is.True(!b)
	c.SetBar("false")
	b, err = c.FnBar().Bool()
	is.NoErr(err)
	is.True(!b)

	c.SetBar("xxx")
	b, err = c.FnBar().Bool()
	is.True(err != nil)
	is.True(!b)

	// float64
	c.SetBar("123.45")
	f, err := c.FnBar().Float64()
	is.NoErr(err)
	expectedF := float64(123.45)
	is.Equal(expectedF, f)
	c.SetBar("xxx")
	f, err = c.FnBar().Float64()
	is.True(err != nil)
	is.Equal(float64(0), f)

	// int64
	c.SetBar("123")
	i, err := c.FnBar().Int64()
	is.NoErr(err)
	expectedI := int64(123)
	is.Equal(expectedI, i)
	c.SetBar("xxx")
	i, err = c.FnBar().Int64()
	is.True(err != nil)
	is.Equal(int64(0), i)

	// string
	s := "This is a string"
	c.SetBar(s)
	is.Equal(s, c.FnBar().String())
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
// However, sprintf does not support named variables,
// and changing the order of variables for _TEMPLATE keys in config files
// must not break previously generated code.
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
	is := testutil.Setup(t)
	params := GetTemplateParams("Fizz{{.Buz}}{{.Meh}}")
	is.Equal([]string{"Buz", "Meh"}, params)
}

func TestGetEnvs(t *testing.T) {
	is := testutil.Setup(t)

	tmp, err := os.MkdirTemp("", "mozey-config")
	is.NoErr(err)
	defer (func() {
		_ = os.RemoveAll(tmp)
	})()

	err = os.WriteFile(
		filepath.Join(tmp, "config.dev.json"),
		[]byte(`{}`),
		0644)
	is.NoErr(err)
	err = os.WriteFile(
		filepath.Join(tmp, "sample.config.dev.json"),
		[]byte(`{}`),
		0644)
	is.NoErr(err)
	err = os.WriteFile(
		filepath.Join(tmp, "config.prod.json"),
		[]byte(`{}`),
		0644)
	is.NoErr(err)
	err = os.WriteFile(
		filepath.Join(tmp, "sample.config.prod.json"),
		[]byte(`{}`),
		0644)
	is.NoErr(err)
	err = os.WriteFile(
		filepath.Join(tmp, "config.stage-ec2.json"),
		[]byte(`{}`),
		0644)
	is.NoErr(err)
	err = os.WriteFile(
		filepath.Join(tmp, "sample.config.stage-ec2.json"),
		[]byte(`{}`),
		0644)
	is.NoErr(err)

	envs, err := getEnvs(tmp, false)
	is.NoErr(err)
	is.Equal([]string{"dev", "prod", "stage-ec2"}, envs)

	envs, err = getEnvs(tmp, true)
	is.NoErr(err)
	is.Equal([]string{"sample.dev", "sample.prod", "sample.stage-ec2"}, envs)
}
