package cmdconfig

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/template"
	"unicode"
)

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
	// Generate config helper
	Generate *string
	// Config file for Env
	Config *Config
	CSV    *bool
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
		c.Keys = append(c.Keys, strings.ToUpper(k))
	}
	// Sort keys
	sort.Strings(c.Keys)
}

// TODO Remove prefix param?
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

type TemplateKey struct {
	KeyPrefix  string
	KeyPrivate string
	Key        string
}

type TemplateData struct {
	Prefix string
	AppDir string
	Keys   []TemplateKey
}

func ToPrivate(str string) string {
	for i, v := range str {
		return string(unicode.ToLower(v)) + str[i+1:]
	}
	return ""
}

func GenerateHelper(in *CmdIn) (buf *bytes.Buffer, err error) {
	// Create template
	t := template.Must(template.New("configTemplate").Parse(configTemplate))

	// Setup template data
	data := TemplateData{
		Prefix: *in.Prefix,
		AppDir: in.AppDir,
	}
	keys := make([]string, len(in.Config.Keys))
	copy(keys, in.Config.Keys)
	keys = append(keys, fmt.Sprintf("%vDIR", *in.Prefix))
	for _, keyPrefix := range keys {
		key := strings.Replace(keyPrefix, *in.Prefix, "", 1)
		key = strings.Replace(key, "_", " ", -1)
		key = strings.ToLower(key)
		key = strings.Replace(strings.Title(key), " ", "", -1)
		templateKey := TemplateKey{
			KeyPrefix:  keyPrefix,
			KeyPrivate: ToPrivate(key),
			Key:        key,
		}
		data.Keys = append(data.Keys, templateKey)
	}

	// Execute the template
	buf = new(bytes.Buffer)
	err = t.Execute(buf, &data)
	if err != nil {
		b, _ := json.MarshalIndent(data, "", "    ")
		fmt.Printf("template data %s %v", "\n", string(b))
		return buf, errors.WithStack(err)
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
		key = strings.ToUpper(key)

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
	_, err = buf.WriteString(strings.Join(a, ","))
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

func Cmd(in *CmdIn) (out *CmdOut, err error) {
	out = &CmdOut{}

	if *in.CSV {
		// Get env CSV
		buf, err := CSV(in)
		if err != nil {
			return out, err
		}
		out.Cmd = "csv"
		out.Buf = buf
		return out, nil

	} else if *in.Compare != "" {
		// Compare keys
		buf, err := CompareKeys(in)
		if err != nil {
			return out, err
		}
		out.Cmd = "compare"
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
		out.Cmd = "generate"
		out.Buf = buf
		return out, nil

	} else if *in.Base64 {
		buf, err := Base64(in)
		if err != nil {
			return out, err
		}
		out.Cmd = "base64"
		out.Buf = buf
		return out, nil

	} else if len(*in.Keys) > 0 {
		// Update config key value pairs
		buf, err := UpdateConfig(in)
		if err != nil {
			return out, err
		}
		out.Cmd = "update_config"
		out.Buf = buf
		return out, nil
	}

	// Default
	// Print set env commands
	buf, err := SetEnv(in)
	if err != nil {
		return out, err
	}
	out.Cmd = "set_env"
	out.Buf = buf
	return out, nil
}

func ParseFlags() *CmdIn {
	in := CmdIn{}

	// Flags
	in.Prefix = flag.String("prefix", "APP_", "Config key prefix")
	in.Env = flag.String("env", "dev", "Config file to use")
	// Default must be empty
	in.Compare = flag.String("compare", "", "Compare config file keys")
	in.Keys = &ArgMap{}
	flag.Var(in.Keys, "key", "Set key and print config JSON")
	in.Values = &ArgMap{}
	flag.Var(in.Values, "value", "Value for last key specified")
	// Default must be empty
	in.Generate = flag.String("generate", "", "Generate config helper at path")
	in.CSV = flag.Bool(
		"csv", false, "Print env key=value CSV")
	in.DryRun = flag.Bool(
		"dry-run", false, "Don't write files, just print result")
	in.Base64 = flag.Bool(
		"base64", false, "Encode config file as base64 string")

	flag.Parse()

	return &in
}

//func fixLineEndings(s string) string {
//	if runtime.GOOS == "windows" {
//		return strings.ReplaceAll(s, "\n", LineBreak)
//	}
//	return s
//}

func (in *CmdIn) Process(out *CmdOut) {
	var err error
	switch out.Cmd {
	case "set_env":
		// Print set and unset env commands
		fmt.Print(out.Buf.String())
		os.Exit(out.ExitCode)

	case "update_config":
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

	case "generate":
		if *in.DryRun {
			fmt.Println(out.Buf.String())
		} else {
			// Write config helper
			err = ioutil.WriteFile(
				filepath.Join(in.AppDir, *in.Generate, "config.go"),
				out.Buf.Bytes(),
				0644)
			if err != nil {
				log.Fatal().Stack().Err(err).Msg("")
			}
		}
		os.Exit(out.ExitCode)

	case "compare":
		fmt.Print(out.Buf.String())
		os.Exit(out.ExitCode)

	case "csv":
		fmt.Print(out.Buf.String())
		os.Exit(out.ExitCode)

	case "base64":
		fmt.Print(out.Buf.String())
		os.Exit(out.ExitCode)
	}
}

// Main can be executed as the default.
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

// standard way to recognize machine-generated files
// https://github.com/golang/go/issues/13560#issuecomment-276866852
var configTemplate = `
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
