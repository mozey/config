package cmdconfig

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"text/template"
	"unicode"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

const CmdBase64 = "base64"
const CmdCompare = "compare"
const CmdCSV = "csv"
const CmdDryRun = "dry-run"
const CmdGenerate = "generate"
const CmdSetEnv = "set_env"
const CmdGet = "get"
const CmdUpdateConfig = "update_config"

// ArgMap for parsing flags with multiple keys
type ArgMap []string

func (a *ArgMap) String() string {
	return strings.Join(*a, ", ")
}
func (a *ArgMap) Set(value string) error {
	*a = append(*a, value)
	return nil
}

// Config file attributes
type Config struct {
	// Map of key to value
	Map map[string]string
	// Keys sorted
	Keys []string
}

// CmdIn for use with command functions
type CmdIn struct {
	// AppDir is the application root
	AppDir string
	// Prefix for env vars
	Prefix *string
	// Env selects the config file
	Env *string
	// Compare config file keys
	Compare *string
	// Readers make testing easier
	ConfigReader  io.Reader
	CompareReader io.Reader
	// Keys to update
	Keys *ArgMap
	// Value to update
	Values *ArgMap
	// PrintValue for the given key
	PrintValue *string
	// Generate config helper
	Generate *string
	// Config file for Env
	Config *Config
	CSV    *bool
	Sep    *string
	DryRun *bool
	// Base64 encode config file
	Base64 *bool
}

// CmdOut for use with Cmd function
type CmdOut struct {
	// Cmd is the unique command that was executed
	Cmd string
	// ExitCode can be non-zero if the err returned is nil
	ExitCode int
	// Buf of cmd output
	Buf *bytes.Buffer
}

// GetPath to config file
func GetPath(appDir string, env string) (string, error) {
	if _, err := os.Stat(appDir); err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf(
				"app dir does not exist %v", appDir)
		} else {
			return "", errors.WithStack(err)
		}
	}

	// Strip "sample." prefix from env
	samplePrefix := "sample."
	sample := ""
	if strings.Contains(env, samplePrefix) {
		sample = samplePrefix
		env = strings.Replace(env, samplePrefix, "", 1)
	}

	return filepath.Join(appDir, fmt.Sprintf("%vconfig.%v.json", sample, env)), nil
}

func RefreshKeys(c *Config) {
	c.Keys = nil
	// Set config keys
	for k := range c.Map {
		c.Keys = append(c.Keys, k)
	}
	// Sort keys
	sort.Strings(c.Keys)
}

// NewConfig reads a config file and sets the key map
func NewConfig(appDir string, env string, prefix string) (c *Config, err error) {
	// Read config file
	configPath, err := GetPath(appDir, env)
	if err != nil {
		return c, err
	}
	b, err := ioutil.ReadFile(configPath)
	if err != nil {
		//log.Debug().Msgf("reading config at path %v", configPath)
		return c, errors.WithStack(err)
	}

	c = &Config{}

	// Unmarshal config.
	// The config file must have a flat key value structure
	err = json.Unmarshal(b, &c.Map)
	if err != nil {
		//log.Debug().Msgf("unmarshal config at path %v", configPath)
		return c, errors.WithStack(err)
	}

	RefreshKeys(c)

	return c, nil
}

// CompareKeys for config files
func CompareKeys(in *CmdIn) (buf *bytes.Buffer, err error) {
	buf = new(bytes.Buffer)

	compConfig, err := NewConfig(in.AppDir, *in.Compare, *in.Prefix)
	if err != nil {
		return buf, err
	}

	unmatched := make([]string, 0, len(in.Config.Keys)+len(compConfig.Keys))

	// Compare config keys
	for _, item := range in.Config.Keys {
		if _, ok := compConfig.Map[item]; !ok {
			unmatched = append(unmatched, item)
		}
	}
	for _, item := range compConfig.Keys {
		if _, ok := in.Config.Map[item]; !ok {
			unmatched = append(unmatched, item)
		}
	}

	// Add unmatched keys to buffer
	sort.Strings(unmatched)
	for _, item := range unmatched {
		buf.WriteString(fmt.Sprintf("%s%s", item, "\n"))
	}

	return buf, nil
}

type GenerateKey struct {
	KeyPrefix  string
	KeyPrivate string
	Key        string
}

type TemplateParam struct {
	KeyPrivate string
	Key        string
	// Implicit is set if the param is also a config key
	Implicit bool
}

// TemplateKey, e.g. APP_TEMPLATE_*
type TemplateKey struct {
	GenerateKey
	ExplicitParams string
	Params         []TemplateParam
}

type GenerateData struct {
	Prefix       string
	AppDir       string
	Keys         []GenerateKey
	TemplateKeys []TemplateKey
	// KeyMap can be used to lookup an index in Keys given a key
	KeyMap map[string]int
}

func NewGenerateData(in *CmdIn) (data GenerateData) {
	// Init
	data = GenerateData{
		Prefix: *in.Prefix,
		AppDir: in.AppDir,
	}

	// APP_DIR is usually not set in the config.json file
	keys := make([]string, len(in.Config.Keys))
	copy(keys, in.Config.Keys)
	keys = append(keys, fmt.Sprintf("%vDIR", *in.Prefix))

	data.Keys = make([]GenerateKey, len(keys))
	data.TemplateKeys = make([]TemplateKey, 0)
	data.KeyMap = make(map[string]int)

	configFileKeys := make(map[string]bool)
	templateKeys := make([]GenerateKey, 0)

	// Prepare data for generating config helper files
	for i, keyWithPrefix := range keys {
		formattedKey := FormatKey(*in.Prefix, keyWithPrefix)
		configFileKeys[formattedKey] = true
		generateKey := GenerateKey{
			KeyPrefix:  keyWithPrefix,
			KeyPrivate: ToPrivate(formattedKey),
			Key:        formattedKey,
		}
		data.Keys[i] = generateKey
		data.KeyMap[formattedKey] = i

		// If template key then append to templateKeys
		if strings.Contains(keyWithPrefix, "_TEMPLATE") {
			templateKeys = append(templateKeys, generateKey)
		}
	}

	// Template keys are use to generate template.go
	for _, generateKey := range templateKeys {
		templateKey := TemplateKey{
			GenerateKey: generateKey,
		}
		params := GetTemplateParams(in.Config.Map[generateKey.KeyPrefix])
		explicitParams := make([]string, 0)
		for _, param := range params {
			keyPrivate := ToPrivate(param)
			implicit := false
			if _, ok := configFileKeys[param]; ok {
				implicit = true
			} else {
				explicitParams = append(explicitParams, keyPrivate)
			}
			templateKey.Params = append(
				templateKey.Params, TemplateParam{
					KeyPrivate: keyPrivate,
					Key:        param,
					Implicit:   implicit,
				})
		}
		if len(explicitParams) > 0 {
			templateKey.ExplicitParams =
				strings.Join(explicitParams, ", ") + " string"
		}
		data.TemplateKeys = append(data.TemplateKeys, templateKey)
	}

	return data
}

// GetTemplateParams from template, e.g.
// passing in "Fizz{{.Buz}}{{.Meh}}" should return ["Buz", "Meh"]
func GetTemplateParams(value string) (params []string) {
	params = make([]string, 0)
	// Replace all mustache params with substring groups match
	s := "\\{\\{\\.(\\w*)}}"
	r, err := regexp.Compile(s)
	if err != nil {
		return params
	}

	matches := r.FindAllStringSubmatch(value, -1)
	for _, match := range matches {
		params = append(params, match[1])
	}

	return params
}

// FormatKey removes the prefix and converts env var to golang var,
// e.g. APP_FOO_BAR becomes FooBar
func FormatKey(prefix, keyWithPrefix string) string {
	key := strings.Replace(keyWithPrefix, prefix, "", 1)
	key = strings.Replace(key, "_", " ", -1)
	key = strings.ToLower(key)
	key = strings.Replace(strings.Title(key), " ", "", -1)
	return key
}

// ToPrivate lowercases the first character of str
func ToPrivate(str string) string {
	for i, v := range str {
		return string(unicode.ToLower(v)) + str[i+1:]
	}
	return ""
}

// GenerateHelper generates helper files, config.go, template.go, etc.
// These files can then be included by users in their own projects
// when they import the config package at the path as per the "generate" flag
func GenerateHelper(in *CmdIn) (buf *bytes.Buffer, err error) {
	// Generate data for executing template
	data := NewGenerateData(in)

	// Initialize return buf,
	// for dry-run it will contain the generated text,
	// otherwise it will contain paths to files that were written
	buf = new(bytes.Buffer)

	// Generate config.go from template
	filePath := filepath.Join(in.AppDir, *in.Generate, "config.go")
	t := template.Must(template.New("generateConfig").Parse(generateConfig))
	generatedBuf := new(bytes.Buffer)
	err = t.Execute(generatedBuf, &data)
	if err != nil {
		b, _ := json.MarshalIndent(data, "", "    ")
		fmt.Printf("template data %s %v", "\n", string(b))
		return generatedBuf, errors.WithStack(err)
	}
	if *in.DryRun {
		// Write file path and generated text to return buf
		buf.WriteString("\n")
		buf.WriteString(fmt.Sprintf("// FilePath: %s", filePath))
		buf.Write(generatedBuf.Bytes())
	} else {
		// Update config.go
		err = ioutil.WriteFile(
			filePath,
			generatedBuf.Bytes(),
			0644)
		if err != nil {
			log.Fatal().Stack().Err(err).Msg("")
		}
		// Write file path to return buf
		buf.WriteString(filePath)
	}

	// Generate template.go from template
	filePath = filepath.Join(in.AppDir, *in.Generate, "template.go")
	t = template.Must(template.New("generateTemplate").Parse(generateTemplate))
	generatedBuf = new(bytes.Buffer)
	err = t.Execute(generatedBuf, &data)
	if err != nil {
		b, _ := json.MarshalIndent(data, "", "    ")
		fmt.Printf("template data %s %v", "\n", string(b))
		return generatedBuf, errors.WithStack(err)
	}
	if *in.DryRun {
		// Write file path and generated text to return buf
		buf.WriteString("\n")
		buf.WriteString(fmt.Sprintf("// FilePath: %s", filePath))
		buf.Write(generatedBuf.Bytes())
	} else {
		// Write template.go
		err = ioutil.WriteFile(
			filePath,
			generatedBuf.Bytes(),
			0644)
		if err != nil {
			log.Fatal().Stack().Err(err).Msg("")
		}
		// Write file path to return buf
		buf.WriteString("\n")
		buf.WriteString(filePath)
	}

	// Generate fn.go from template
	filePath = filepath.Join(in.AppDir, *in.Generate, "fn.go")
	t = template.Must(template.New("generateFn").Parse(generateFn))
	generatedBuf = new(bytes.Buffer)
	err = t.Execute(generatedBuf, &data)
	if err != nil {
		b, _ := json.MarshalIndent(data, "", "    ")
		fmt.Printf("template data %s %v", "\n", string(b))
		return generatedBuf, errors.WithStack(err)
	}
	if *in.DryRun {
		// Write file path and generated text to return buf
		buf.WriteString("\n")
		buf.WriteString(fmt.Sprintf("// FilePath: %s", filePath))
		buf.Write(generatedBuf.Bytes())
	} else {
		// Write fn.go
		err = ioutil.WriteFile(
			filePath,
			generatedBuf.Bytes(),
			0644)
		if err != nil {
			log.Fatal().Stack().Err(err).Msg("")
		}
		// Write file path to return buf
		buf.WriteString("\n")
		buf.WriteString(filePath)
	}

	return buf, nil
}

func UpdateConfig(in *CmdIn) (buf *bytes.Buffer, err error) {
	buf = new(bytes.Buffer)

	// Setup existing key value pairs
	m := make(map[string]string)
	for _, key := range in.Config.Keys {
		m[key] = in.Config.Map[key]
	}

	// Validate input
	keys := *in.Keys
	values := *in.Values
	for i, key := range keys {
		if !strings.HasPrefix(key, *in.Prefix) {
			return buf, errors.WithStack(
				fmt.Errorf("key must strart with prefix %v", in.Prefix))
		}

		if i > len(*in.Values)-1 {
			return buf, errors.WithStack(
				fmt.Errorf("missing value for key %v", key))
		}
		value := values[i]

		// Update key value pairs
		//log.Debug().Msgf("Config %v %v=%v", *in.Env, key, value)
		m[key] = value
		RefreshKeys(in.Config)
	}

	// Marshal config JSON
	b, err := json.MarshalIndent(m, "", "    ")
	if err != nil {
		return buf, errors.WithStack(err)
	}
	buf.Write(b)

	return buf, nil
}

type EnvKeys map[string]bool

func SetEnv(in *CmdIn) (buf *bytes.Buffer, err error) {
	// Create map of env vars starting with Prefix
	envKeys := EnvKeys{}
	for _, v := range os.Environ() {
		a := strings.Split(v, "=")
		if len(a) == 2 {
			key := a[0]
			if strings.HasPrefix(key, *in.Prefix) {
				envKeys[a[0]] = true
			}
		}
	}

	buf = new(bytes.Buffer)

	// Commands to set env
	for _, key := range in.Config.Keys {
		buf.WriteString(fmt.Sprintf(ExportFormat, key, in.Config.Map[key]))
		buf.WriteString("\n")
		envKeys[key] = false
	}

	// Don't print command to unset APP_DIR
	// https://github.com/mozey/config/issues/9
	appDirKey := fmt.Sprintf("%vDIR", *in.Prefix)
	if _, ok := envKeys[appDirKey]; ok {
		envKeys[appDirKey] = false
	}

	// Unset env vars not listed in the config file
	for key, unset := range envKeys {
		if unset {
			buf.WriteString(fmt.Sprintf(UnsetFormat, key))
			buf.WriteString("\n")
		}
	}

	return buf, nil
}

func CSV(in *CmdIn) (buf *bytes.Buffer, err error) {
	buf = new(bytes.Buffer)

	a := make([]string, len(in.Config.Keys))
	for i, key := range in.Config.Keys {
		value := in.Config.Map[key]
		if strings.Contains(value, "\n") {
			return buf, errors.WithStack(
				fmt.Errorf("values must not contain newlines"))
		}
		if strings.Contains(value, ",") {
			return buf, errors.WithStack(
				fmt.Errorf("values must not contain commas"))
		}
		a[i] = fmt.Sprintf("%v=%v", key, value)
	}

	// Do not use encoding/csv, the writer will append a newline
	_, err = buf.WriteString(strings.Join(a, *in.Sep))
	if err != nil {
		return buf, errors.WithStack(err)
	}

	return buf, nil
}

func Base64(in *CmdIn) (buf *bytes.Buffer, err error) {
	buf = new(bytes.Buffer)

	b, err := json.Marshal(in.Config.Map)
	if err != nil {
		return buf, errors.WithStack(err)
	}
	encoded := base64.StdEncoding.EncodeToString(b)
	buf.Write([]byte(encoded))

	return buf, nil
}

func PrintValue(in *CmdIn) (buf *bytes.Buffer, err error) {
	buf = new(bytes.Buffer)
	key := *in.PrintValue

	if value, ok := in.Config.Map[key]; ok {
		buf.WriteString(value)
		return buf, nil
	}

	return buf, errors.WithStack(
		fmt.Errorf("missing value for key %v", key))
}

func Cmd(in *CmdIn) (out *CmdOut, err error) {
	out = &CmdOut{}

	if *in.CSV {
		// Get env CSV
		buf, err := CSV(in)
		if err != nil {
			return out, err
		}
		out.Cmd = CmdCSV
		out.Buf = buf
		return out, nil

	} else if *in.Compare != "" {
		// Compare keys
		buf, err := CompareKeys(in)
		if err != nil {
			return out, err
		}
		out.Cmd = CmdCompare
		out.Buf = buf
		if out.Buf.Len() > 0 {
			out.ExitCode = 1
		}
		return out, nil

	} else if *in.Generate != "" {
		// Generate config helper
		buf, err := GenerateHelper(in)
		if err != nil {
			return out, err
		}
		out.Cmd = CmdGenerate
		out.Buf = buf
		return out, nil

	} else if *in.Base64 {
		buf, err := Base64(in)
		if err != nil {
			return out, err
		}
		out.Cmd = CmdBase64
		out.Buf = buf
		return out, nil

	} else if len(*in.Keys) > 0 {
		// Update config key value pairs
		buf, err := UpdateConfig(in)
		if err != nil {
			return out, err
		}
		out.Cmd = CmdUpdateConfig
		out.Buf = buf
		return out, nil

	} else if *in.PrintValue != "" {
		buf, err := PrintValue(in)
		if err != nil {
			return out, err
		}
		out.Cmd = CmdGet
		out.Buf = buf
		return out, nil
	}

	// Default
	// Print set env commands
	buf, err := SetEnv(in)
	if err != nil {
		return out, err
	}
	out.Cmd = CmdSetEnv
	out.Buf = buf
	return out, nil
}

func ParseFlags() *CmdIn {
	in := CmdIn{}

	// Flags
	in.Prefix = flag.String("prefix", "APP_", "Config key prefix")
	in.Env = flag.String("env", "dev", "Config file to use")
	// Default must be empty
	in.Compare = flag.String(CmdCompare, "", "Compare config file keys")
	in.Keys = &ArgMap{}
	flag.Var(in.Keys, "key", "Set key and print config JSON")
	in.Values = &ArgMap{}
	flag.Var(in.Values, "value", "Value for last key specified")
	// Default must be empty
	in.PrintValue = flag.String(CmdGet, "", "Print value for given key")
	// Default must be empty
	in.Generate = flag.String(CmdGenerate, "", "Generate config helper at path")
	in.CSV = flag.Bool(
		CmdCSV, false, "Print env as a list of key=value")
	in.Sep = flag.String("sep", ",", "Separator for with with csv flag")
	in.DryRun = flag.Bool(
		CmdDryRun, false, "Don't write files, just print result")
	in.Base64 = flag.Bool(
		CmdBase64, false, "Encode config file as base64 string")

	flag.Parse()

	return &in
}

func (in *CmdIn) Process(out *CmdOut) {
	switch out.Cmd {
	case CmdSetEnv:
		// Print set and unset env commands
		fmt.Print(out.Buf.String())
		os.Exit(out.ExitCode)

	case CmdGet:
		// Print value for the given key
		fmt.Print(out.Buf.String())
		os.Exit(out.ExitCode)

	case CmdUpdateConfig:
		// Print config
		if *in.DryRun {
			fmt.Println(out.Buf.String())
		} else {
			configPath, err := GetPath(in.AppDir, *in.Env)
			if err != nil {
				log.Fatal().Stack().Err(err).Msg("")
			}
			// Update config file
			info, err := os.Stat(configPath)
			if err != nil {
				log.Fatal().Stack().Err(err).Msg("")
			}
			perm := info.Mode() // Preserve existing mode
			err = ioutil.WriteFile(configPath, out.Buf.Bytes(), perm)
			if err != nil {
				log.Fatal().Stack().Err(err).Msg("")
			}
		}
		os.Exit(out.ExitCode)

	case CmdGenerate:
		fmt.Println(out.Buf.String())
		os.Exit(out.ExitCode)

	case CmdCompare:
		fmt.Print(out.Buf.String())
		os.Exit(out.ExitCode)

	case CmdCSV:
		fmt.Print(out.Buf.String())
		os.Exit(out.ExitCode)

	case CmdBase64:
		fmt.Print(out.Buf.String())
		os.Exit(out.ExitCode)
	}
}

// Main can be executed by default.
// For custom flags and CMDs copy the code below.
// Try not to change the behaviour of default CMDs,
// e.g. custom flags must only add functionality
func Main() {
	// Define custom flags here...

	// Parse flags
	in := ParseFlags()
	prefix := *in.Prefix
	if prefix[len(prefix)-1:] != "_" {
		// Prefix must end with underscore
		*in.Prefix = fmt.Sprintf("%s_", prefix)
	}

	// appDir is required
	appDirKey := fmt.Sprintf("%sDIR", *in.Prefix)
	appDir := os.Getenv(appDirKey)
	if appDir == "" {
		fmt.Printf("%v env not set%s", appDirKey, "\n")
		os.Exit(1)
	}
	in.AppDir = appDir

	// Set config
	config, err := NewConfig(in.AppDir, *in.Env, *in.Prefix)
	if err != nil {
		log.Fatal().
			Str("APP_DIR", in.AppDir).
			Str("env", *in.Env).
			Stack().Err(err).Msg("")
	}
	in.Config = config

	// Run custom commands here...

	// Run cmd
	out, err := Cmd(in)
	if err != nil {
		log.Fatal().Stack().Err(err).Msg("")
	}

	// Process cmd results
	in.Process(out)
}

// generateConfig text template to generate config.go file
// standard way to recognize machine-generated files
// https://github.com/golang/go/issues/13560#issuecomment-276866852
var generateConfig = `
// Code generated with https://github.com/mozey/config DO NOT EDIT

package config

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"io/ioutil"
	"os"
	"path/filepath"
)

{{range .Keys}}
// {{.KeyPrefix}}
var {{.KeyPrivate}} string{{end}}


// Config fields correspond to config file keys less the prefix
type Config struct {
	{{range .Keys}}
	{{.KeyPrivate}} string // {{.KeyPrefix}}{{end}}
}

{{range .Keys}}
// {{.Key}} is {{.KeyPrefix}}
func (c *Config) {{.Key}}() string {
	return c.{{.KeyPrivate}}
}{{end}}

{{range .Keys}}
// Set{{.Key}} overrides the value of {{.KeyPrivate}}
func (c *Config) Set{{.Key}}(v string) {
	c.{{.KeyPrivate}} = v
}
{{end}}

// New creates an instance of Config.
// Build with ldflags to set the package vars.
// Env overrides package vars.
// Fields correspond to the config file keys less the prefix.
// The config file must have a flat structure
func New() *Config {
	conf := &Config{}
	SetVars(conf)
	SetEnv(conf)
	return conf
}

// SetVars sets non-empty package vars on Config
func SetVars(conf *Config) {
	{{range .Keys}}
	if {{.KeyPrivate}} != "" {
		conf.{{.KeyPrivate}} = {{.KeyPrivate}}
	}
	{{end}}
}

// SetEnv sets non-empty env vars on Config
func SetEnv(conf *Config) {
	var v string

	{{range .Keys}}
	v = os.Getenv("{{.KeyPrefix}}")
	if v != "" {
		conf.{{.KeyPrivate}} = v
	}
	{{end}}
}

// GetMap of all env vars
func (c *Config) GetMap() map[string]string {
	m := make(map[string]string)
	{{range .Keys}}
	m["{{.KeyPrefix}}"] = c.{{.KeyPrivate}}
	{{end}}
	return m
}

// SetEnvBase64 decodes and sets env from the given base64 string
func SetEnvBase64(configBase64 string) (err error) {
	// Decode base64
	decoded, err := base64.StdEncoding.DecodeString(configBase64)
	if err != nil {
		return errors.WithStack(err)
	}
	// UnMarshall json
	configMap := make(map[string]string)
	err = json.Unmarshal(decoded, &configMap)
	if err != nil {
		return errors.WithStack(err)
	}
	// Set config
	for key, value := range configMap {
		err = os.Setenv(key, value)
		if err != nil {
			return errors.WithStack(err)
		}
	}

	return nil
}

// LoadFile sets the env from file and returns a new instance of Config
func LoadFile(mode string) (conf *Config, err error) {
	appDir := os.Getenv("{{.Prefix}}DIR")
	p := filepath.Join(appDir, fmt.Sprintf("config.%v.json", mode))
	b, err := ioutil.ReadFile(p)
	if err != nil {
		return nil, err
	}
	configMap := make(map[string]string)
	err = json.Unmarshal(b, &configMap)
	if err != nil {
		return nil, err
	}
	for key, val := range configMap {
		_ = os.Setenv(key, val)
	}
	return New(), nil
}
`

// generateTemplate text template to generate template.go file
var generateTemplate = `
// Code generated with https://github.com/mozey/config DO NOT EDIT

package config

import (
	"bytes"
	"text/template"
)

{{range .TemplateKeys}}
// Exec{{.Key}} fills {{.KeyPrefix}} with the given params
func (c *Config) Exec{{.Key}}({{.ExplicitParams}}) string {
	t := template.Must(template.New("{{.KeyPrivate}}").Parse(c.{{.KeyPrivate}}))
	b := bytes.Buffer{}
	_ = t.Execute(&b, map[string]interface{}{
	{{range .Params}}
		"{{.Key}}": {{if .Implicit}}c.{{end}}{{.KeyPrivate}},{{end}}
	})
	return b.String()
}
{{end}}
`

// generateFn text template to generate fn.go file
var generateFn = `
package config

import (
	"fmt"
	"strconv"
	"strings"
)

type Fn struct {
	input string
	// output of the last function,
	// might be useful when chaining multiple functions?
	output string
}

// .............................................................................
// Methods to set function input

{{range .Keys}}
// Fn{{.Key}} sets the function input to the value of {{.KeyPrefix}}
func (c *Config) Fn{{.Key}}() *Fn {
	fn := Fn{}
	fn.input = c.{{.KeyPrivate}}
	fn.output = ""
	return &fn
}
{{end}}

// .............................................................................
// Type conversion functions

// Bool parses a bool from the value or returns an error.
// Valid values are "1", "0", "true", or "false".
// The value is not case-sensitive
func (fn *Fn) Bool() (bool, error) {
	v := strings.ToLower(fn.input)
	if v == "1" || v == "true" {
		return true, nil
	}
	if v == "0" || v == "false" {
		return false, nil
	}
	return false, fmt.Errorf("invalid value %s", fn.input)
}

// Float64 parses a float64 from the value or returns an error
func (fn *Fn) Float64() (float64, error) {
	f, err := strconv.ParseFloat(fn.input, 64)
	if err != nil {
		return f, err
	}
	return f, nil
}

// Int64 parses an int64 from the value or returns an error
func (fn *Fn) Int64() (int64, error) {
	i, err := strconv.ParseInt(fn.input, 10, 64)
	if err != nil {
		return i, err
	}
	return i, nil
}

// String returns the input as is
func (fn *Fn) String() string {
	return fn.input
}
`
