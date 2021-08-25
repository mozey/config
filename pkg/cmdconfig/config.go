package cmdconfig

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
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

// .............................................................................
// TODO Rethink naming of the constants below

const CmdBase64 = "base64"
const CmdCompare = "compare"
const CmdCSV = "csv"
const CmdDryRun = "dry-run" // TODO This isn't a command as such...
const CmdGenerate = "generate"
const CmdSetEnv = "set-env"
const CmdGet = "get"
const CmdUpdateConfig = "update-config"

// .............................................................................

// ArgMap for parsing flags with multiple keys
type ArgMap []string

func (a *ArgMap) String() string {
	return strings.Join(*a, ", ")
}
func (a *ArgMap) Set(value string) error {
	*a = append(*a, value)
	return nil
}

// .............................................................................

// Config file attributes.
// Note, this is not the same struct as the generated config.Config,
// e.g. pkg/cmdconfig/testdata/config.go
// The latter has properties for each config attribute in a project,
// whereas this type is generic
type Config struct {
	// Map of key to value
	Map map[string]string
	// Keys sorted
	Keys []string
}

// ConfigCache is used to avoid reading the same config file twice,
// use full path to config file as the map key
type ConfigCache map[string]*Config

var configCache ConfigCache

// .............................................................................

// CmdIn for use with command functions
type CmdIn struct {
	// AppDir is the application root
	AppDir string
	// Prefix for env vars
	Prefix string
	// Env selects the config file
	Env string
	// All makes the cmd apply to all config files in APP_DIR, including samples
	// https://github.com/mozey/config/issues/2
	All bool
	// Compare config file keys
	Compare string
	// Keys to update
	Keys ArgMap
	// Value to update
	Values ArgMap
	// PrintValue for the given key
	PrintValue string
	// Generate config helper
	Generate string
	// Config file for Env
	Config *Config
	CSV    bool
	Sep    string
	DryRun bool
	// Base64 encode config file
	Base64 bool
}

// .............................................................................

type File struct {
	// Path to file
	Path string
	// Buf for new file content
	Buf *bytes.Buffer
}

type Files []File

// CmdOut for use with Cmd function
type CmdOut struct {
	// Cmd is the unique command that was executed
	Cmd string
	// ExitCode can be non-zero if the err returned is nil
	ExitCode int
	// Buf of cmd output
	Buf *bytes.Buffer
	// Files to write if in.DryRun is not set
	Files Files
}

// .............................................................................

// GetEnvs globs all config files in APP_DIR to list possible values of env
func GetEnvs(appDir string, includeSamples bool) (envs []string, err error) {
	envs = make([]string, 0)

	// Find matching files
	fileNamePattern := "config.*.json"
	if includeSamples {
		fileNamePattern = "sample.config.*.json"
	}
	pattern := filepath.Join(appDir, fileNamePattern)
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return envs, errors.WithStack(err)
	}

	// Regexp to submatch env from file name
	s := "config\\.(\\w*)\\.json"
	r, err := regexp.Compile(s)
	if err != nil {
		return envs, errors.WithStack(err)
	}

	for _, match := range matches {
		baseName := filepath.Base(match)
		matches := r.FindStringSubmatch(baseName)
		if len(matches) == 2 {
			env := matches[1]
			if strings.HasPrefix(baseName, SamplePrefix) {
				env = fmt.Sprintf("%s%s", SamplePrefix, env)
			}
			envs = append(envs, env)
		}
	}
	return envs, nil
}

const SamplePrefix = "sample."

// GetConfigFilePath returns the path to a config file.
// It can also be used to return the path to a sample config file by prefixing env.
// For example, to get the path to "sample.config.dev.json" pass env="sample.dev"
func GetConfigFilePath(appDir string, env string) (string, error) {
	if _, err := os.Stat(appDir); err != nil {
		if os.IsNotExist(err) {
			return "", errors.WithStack(fmt.Errorf(
				"app dir does not exist %v", appDir))
		} else {
			return "", errors.WithStack(err)
		}
	}

	// Strip SamplePrefix from env
	sample := ""
	if strings.Contains(env, SamplePrefix) {
		sample = SamplePrefix
		env = strings.Replace(env, SamplePrefix, "", 1)
	}

	return filepath.Join(
		appDir, fmt.Sprintf("%vconfig.%v.json", sample, env)), nil
}

// TODO This should be a method on Config?
func RefreshKeys(c *Config) {
	c.Keys = nil
	// Set config keys
	for k := range c.Map {
		c.Keys = append(c.Keys, k)
	}
	// Sort keys
	sort.Strings(c.Keys)
}

// NewConfig reads a config file and sets the key map.
// If env is set on ConfigCache, use it, and avoid reading the file again
func NewConfig(appDir string, env string) (configPath string, c *Config, err error) {
	configPath, err = GetConfigFilePath(appDir, env)
	if err != nil {
		return configPath, c, err
	}

	// Cached?
	if configCache == nil {
		// Init cache
		configCache = make(ConfigCache)
	} else {
		var ok bool
		if c, ok = configCache[configPath]; ok {
			// Return cached config
			return configPath, c, nil
		}
	}

	// New config
	c = &Config{}

	// Read config file
	b, err := ioutil.ReadFile(configPath)
	if err != nil {
		return configPath, c, errors.WithStack(err)
	}

	// Unmarshal config.
	// The config file must have a flat key value structure
	err = json.Unmarshal(b, &c.Map)
	if err != nil {
		return configPath, c, errors.WithStack(err)
	}

	RefreshKeys(c)

	// Add to cache
	configCache[configPath] = c

	return configPath, c, nil
}

// .............................................................................

// compareKeys for config files
func compareKeys(in *CmdIn) (buf *bytes.Buffer, files []File, err error) {
	buf = new(bytes.Buffer)

	_, compConfig, err := NewConfig(in.AppDir, in.Compare)
	if err != nil {
		return buf, files, err
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

	return buf, files, nil
}

// .............................................................................
// TODO Move this to pkg/cmdconfig/generate.go

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

func NewGenerateData(in *CmdIn) (data *GenerateData) {
	// Init
	data = &GenerateData{
		Prefix: in.Prefix,
		AppDir: in.AppDir,
	}

	// APP_DIR is usually not set in the config.json file
	keys := make([]string, len(in.Config.Keys))
	copy(keys, in.Config.Keys)
	keys = append(keys, fmt.Sprintf("%vDIR", in.Prefix))

	data.Keys = make([]GenerateKey, len(keys))
	data.TemplateKeys = make([]TemplateKey, 0)
	data.KeyMap = make(map[string]int)

	configFileKeys := make(map[string]bool)
	templateKeys := make([]GenerateKey, 0)

	// Prepare data for generating config helper files
	for i, keyWithPrefix := range keys {
		formattedKey := FormatKey(in.Prefix, keyWithPrefix)
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

// executeTemplate executes the template for the specified file name and data
func executeTemplate(in *CmdIn, fileName string, data *GenerateData) (
	filePath string, buf *bytes.Buffer, err error) {

	filePath = filepath.Join(in.AppDir, in.Generate, fileName)
	textTemplate, err := GetTemplate(fileName)
	if err != nil {
		return filePath, buf, err
	}
	t := template.Must(
		template.New(fmt.Sprintf("generate%s", fileName)).Parse(textTemplate))
	buf = new(bytes.Buffer)
	err = t.Execute(buf, &data)
	if err != nil {
		return filePath, buf, errors.WithStack(err)
	}
	return filePath, buf, nil
}

// generateHelpers generates helper files, config.go, template.go, etc.
// These files can then be included by users in their own projects
// when they import the config package at the path as per the "generate" flag
func generateHelpers(in *CmdIn) (buf *bytes.Buffer, files []File, err error) {
	// Generate data for executing template
	data := NewGenerateData(in)

	// NOTE buf is usually filled with content to be written to stdout.
	// For the generate flag the contents of buf depends on the dry run flag,
	// and that is checked elsewhere

	// files contains file paths and generated code,
	// depending on the dry run flag it may be written
	// to stdout or the file system
	files = make([]File, 3)

	filePath, buf, err := executeTemplate(in, FileNameConfigGo, data)
	if err != nil {
		return buf, files, err
	}
	files[0] = File{
		Path: filePath,
		Buf:  bytes.NewBuffer(buf.Bytes()),
	}

	filePath, buf, err = executeTemplate(in, FileNameTemplateGo, data)
	if err != nil {
		return buf, files, err
	}
	files[1] = File{
		Path: filePath,
		Buf:  bytes.NewBuffer(buf.Bytes()),
	}

	filePath, buf, err = executeTemplate(in, FileNameFnGo, data)
	if err != nil {
		return buf, files, err
	}
	files[2] = File{
		Path: filePath,
		Buf:  bytes.NewBuffer(buf.Bytes()),
	}

	return buf, files, nil
}

// .............................................................................

// refreshConfigByEnv replaces the given key value pairs in the specified env,
// and returns sorted JSON that can be used to replace the config file contents
func refreshConfigByEnv(appDir string, prefix string, env string, keys ArgMap, values ArgMap) (
	configPath string, b []byte, err error) {

	// Read config for the given env from file, or load from cache
	configPath, conf, err := NewConfig(appDir, env)
	if err != nil {
		return configPath, b, err
	}

	// Setup existing key value pairs
	m := make(map[string]string)
	for _, key := range conf.Keys {
		m[key] = conf.Map[key]
	}

	// Validate input
	for i, key := range keys {
		if !strings.HasPrefix(key, prefix) {
			return configPath, b, errors.WithStack(
				fmt.Errorf(
					"key for env %s must strart with prefix %s", env, prefix))
		}

		if i > len(values)-1 {
			return configPath, b, errors.WithStack(
				fmt.Errorf("env %s missing value for key %s", env, key))
		}
		value := values[i]

		// Update key value pairs
		m[key] = value
		RefreshKeys(conf)
	}

	// Marshal config JSON
	b, err = json.MarshalIndent(m, "", "    ")
	if err != nil {
		return configPath, b, errors.WithStack(err)
	}

	return configPath, b, nil
}

func updateConfig(in *CmdIn) (buf *bytes.Buffer, files []File, err error) {
	buf = new(bytes.Buffer)
	var b []byte

	if in.All {
		// TODO If in.All is set then use GetEnvs
		// to call updateConfigByEnv for all envs
		// files = make([]File, 3)
	} else if in.Env == "*" {
	} else if in.Env == "sample.*" {

	} else {
		files = make([]File, 1)
		var configPath string
		configPath, b, err = refreshConfigByEnv(
			in.AppDir, in.Prefix, in.Env, in.Keys, in.Values)
		if err != nil {
			return buf, files, err
		}
		files[0] = File{
			Path: configPath,
			Buf:  bytes.NewBuffer(b),
		}
	}

	return buf, files, nil
}

// .............................................................................

type EnvKeys map[string]bool

func setEnv(in *CmdIn) (buf *bytes.Buffer, files []File, err error) {
	// Create map of env vars starting with Prefix
	envKeys := EnvKeys{}
	for _, v := range os.Environ() {
		a := strings.Split(v, "=")
		if len(a) == 2 {
			key := a[0]
			if strings.HasPrefix(key, in.Prefix) {
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
	appDirKey := fmt.Sprintf("%vDIR", in.Prefix)
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

	return buf, files, nil
}

// .............................................................................

func generateCSV(in *CmdIn) (buf *bytes.Buffer, files []File, err error) {
	buf = new(bytes.Buffer)

	a := make([]string, len(in.Config.Keys))
	for i, key := range in.Config.Keys {
		value := in.Config.Map[key]
		if strings.Contains(value, "\n") {
			return buf, files, errors.WithStack(
				fmt.Errorf("values must not contain newlines"))
		}
		if strings.Contains(value, ",") {
			return buf, files, errors.WithStack(
				fmt.Errorf("values must not contain commas"))
		}
		a[i] = fmt.Sprintf("%v=%v", key, value)
	}

	// Do not use encoding/csv, the writer will append a newline
	_, err = buf.WriteString(strings.Join(a, in.Sep))
	if err != nil {
		return buf, files, errors.WithStack(err)
	}

	return buf, files, nil
}

// .............................................................................

func encodeBase64(in *CmdIn) (buf *bytes.Buffer, files []File, err error) {
	buf = new(bytes.Buffer)

	b, err := json.Marshal(in.Config.Map)
	if err != nil {
		return buf, files, errors.WithStack(err)
	}
	encoded := base64.StdEncoding.EncodeToString(b)
	buf.Write([]byte(encoded))

	return buf, files, nil
}

// .............................................................................

func printValue(in *CmdIn) (buf *bytes.Buffer, files []File, err error) {
	buf = new(bytes.Buffer)
	key := in.PrintValue

	if value, ok := in.Config.Map[key]; ok {
		buf.WriteString(value)
		return buf, files, nil
	}

	return buf, files, errors.WithStack(
		fmt.Errorf("missing value for key %v", key))
}

// .............................................................................
// TODO Move this to pkg/cmdconfig/cmd.go

// Cmd runs a command given flags and input from the user
func Cmd(in *CmdIn) (out *CmdOut, err error) {
	out = &CmdOut{}

	if in.CSV {
		// Generate CSV from env
		buf, files, err := generateCSV(in)
		if err != nil {
			return out, err
		}
		out.Cmd = CmdCSV
		out.Buf = buf
		out.Files = files
		return out, nil

	} else if in.Compare != "" {
		// Compare keys
		buf, files, err := compareKeys(in)
		if err != nil {
			return out, err
		}
		out.Cmd = CmdCompare
		out.Buf = buf
		if out.Buf.Len() > 0 {
			out.ExitCode = 1
		}
		out.Files = files
		return out, nil

	} else if in.Generate != "" {
		// Generate config helper
		buf, files, err := generateHelpers(in)
		if err != nil {
			return out, err
		}
		out.Cmd = CmdGenerate
		out.Buf = buf
		out.Files = files
		return out, nil

	} else if in.Base64 {
		buf, files, err := encodeBase64(in)
		if err != nil {
			return out, err
		}
		out.Cmd = CmdBase64
		out.Buf = buf
		out.Files = files
		return out, nil

	} else if len(in.Keys) > 0 {
		// Update config key value pairs
		buf, files, err := updateConfig(in)
		if err != nil {
			return out, err
		}
		out.Cmd = CmdUpdateConfig
		out.Buf = buf
		out.Files = files
		return out, nil

	} else if in.PrintValue != "" {
		buf, files, err := printValue(in)
		if err != nil {
			return out, err
		}
		out.Cmd = CmdGet
		out.Buf = buf
		out.Files = files
		return out, nil
	}

	// Default
	// Print set env commands
	buf, files, err := setEnv(in)
	if err != nil {
		return out, err
	}
	out.Cmd = CmdSetEnv
	out.Buf = buf
	out.Files = files
	return out, nil
}

// .............................................................................
// TODO Move this to pkg/cmdconfig/main.go

// ParseFlags before calling Cmd
func ParseFlags() *CmdIn {
	in := CmdIn{}

	// Flags
	flag.StringVar(&in.Prefix, "prefix", "APP_", "Config key prefix")
	flag.StringVar(&in.Env, "env", "dev",
		"Config file to use, also supports wildcards * and sample.*")
	flag.BoolVar(&in.All, "all", false, "Apply to all config files and samples")
	// Default must be empty
	flag.StringVar(&in.Compare, CmdCompare, "", "Compare config file keys")
	in.Keys = ArgMap{}
	flag.Var(&in.Keys, "key", "Set key and print config JSON")
	in.Values = ArgMap{}
	flag.Var(&in.Values, "value", "Value for last key specified")
	// Default must be empty
	flag.StringVar(&in.PrintValue, CmdGet, "", "Print value for given key")
	// Default must be empty
	flag.StringVar(&in.Generate,
		CmdGenerate, "", "Generate config helper at path")
	flag.BoolVar(&in.CSV,
		CmdCSV, false, "Print env as a list of key=value")
	flag.StringVar(&in.Sep, "sep", ",", "Separator for use with csv flag")
	flag.BoolVar(&in.DryRun,
		CmdDryRun, false, "Don't write files, just print result")
	flag.BoolVar(&in.Base64,
		CmdBase64, false, "Encode config file as base64 string")

	flag.Parse()

	return &in
}

// .............................................................................
// TODO Move this to pkg/cmdconfig/cmd.go

// Process the output of the Cmd func.
// For example, this is where results are printed to stdout or disk IO happens,
// depending on the whether the in.DryRun flag was set
func (in *CmdIn) Process(out *CmdOut) {
	switch out.Cmd {
	case CmdSetEnv:
		// .....................................................................
		// Print set and unset env commands
		fmt.Print(out.Buf.String())
		os.Exit(out.ExitCode)

	case CmdGet:
		// .....................................................................
		// Print value for the given key
		fmt.Print(out.Buf.String())
		os.Exit(out.ExitCode)

	case CmdUpdateConfig:
		// .....................................................................
		if in.DryRun {
			// If there is only one config file to update,
			// then print the "new" contents
			if len(out.Files) == 1 {
				fmt.Println(out.Files[0].Buf.String())
			} else {
				// Otherwise print file paths and contents
				for _, file := range out.Files {
					out.Buf.WriteString("\n")
					out.Buf.WriteString(fmt.Sprintf("// FilePath: %s", file.Path))
					out.Buf.Write(file.Buf.Bytes())
				}
			}
		} else {
			for _, file := range out.Files {
				// Create or update the file
				err := os.WriteFile(
					file.Path, file.Buf.Bytes(), 0644)
				if err != nil {
					err = errors.WithStack(err)
					log.Error().
						Str("file_path", file.Path).
						Stack().Err(err)
					os.Exit(1)
				}
				// Print file path only
				out.Buf.WriteString(file.Path)
				out.Buf.WriteString("\n")
			}
		}
		os.Exit(out.ExitCode)

	case CmdGenerate:
		// .....................................................................
		if out.Buf == nil {
			out.Buf = new(bytes.Buffer)
		}
		if in.DryRun {
			// Print file paths and generated text
			for _, file := range out.Files {
				out.Buf.WriteString("\n")
				out.Buf.WriteString(fmt.Sprintf("// FilePath: %s", file.Path))
				out.Buf.Write(file.Buf.Bytes())
			}
		} else {
			for _, file := range out.Files {
				// Create or update the file
				err := os.WriteFile(
					file.Path, file.Buf.Bytes(), 0644)
				if err != nil {
					err = errors.WithStack(err)
					log.Error().
						Str("file_path", file.Path).
						Stack().Err(err)
					os.Exit(1)
				}
				// Print file path only
				out.Buf.WriteString(file.Path)
				out.Buf.WriteString("\n")
			}
		}
		fmt.Println(out.Buf.String())
		os.Exit(out.ExitCode)

	case CmdCompare:
		// .....................................................................
		fmt.Print(out.Buf.String())
		os.Exit(out.ExitCode)

	case CmdCSV:
		// .....................................................................
		fmt.Print(out.Buf.String())
		os.Exit(out.ExitCode)

	case CmdBase64:
		// .....................................................................
		fmt.Print(out.Buf.String())
		os.Exit(out.ExitCode)
	}
}

// .............................................................................
// TODO Move this to pkg/cmdconfig/main.go

// Main can be executed by default.
// For custom flags and CMDs copy the code below.
// Try not to change the behaviour of default CMDs,
// e.g. custom flags must only add functionality
func Main() {
	// Define custom flags here...

	// Parse flags
	in := ParseFlags()
	prefix := in.Prefix
	if prefix[len(prefix)-1:] != "_" {
		// Prefix must end with underscore
		in.Prefix = fmt.Sprintf("%s_", prefix)
	}

	// appDir is required
	appDirKey := fmt.Sprintf("%sDIR", in.Prefix)
	appDir := os.Getenv(appDirKey)
	if appDir == "" {
		err := fmt.Errorf("%v env not set%s", appDirKey, "\n")
		log.Error().Stack().Err(err).Msg("")
		os.Exit(1)
	}
	in.AppDir = appDir

	// Set config
	configPath, config, err := NewConfig(in.AppDir, in.Env)
	if err != nil {
		log.Error().
			Str("config_path", configPath).
			Stack().Err(err).Msg("")
		os.Exit(1)
	}
	in.Config = config

	// Run custom commands here...

	// Run cmd
	out, err := Cmd(in)
	if err != nil {
		log.Error().Stack().Err(err).Msg("")
		os.Exit(1)
	}

	// Process cmd results
	in.Process(out)
}

// .............................................................................
// Template funcs

// FileNameConfigGo for config.go
const FileNameConfigGo = "config.go"

// FileNameTemplateGo for template.go
const FileNameTemplateGo = "template.go"

// FileNameFnGo for fn.go
const FileNameFnGo = "fn.go"

// GetTemplate returns the text template for the given file name.
func GetTemplate(fileName string) (s string, err error) {
	if fileName == FileNameConfigGo {
		return templateConfigGo, nil
	}

	if fileName == FileNameTemplateGo {
		return templateTemplateGo, nil
	}

	if fileName == FileNameFnGo {
		return templateFnGo, nil
	}

	return s, errors.WithStack(
		fmt.Errorf("invalid file name %s", fileName))
}

// .............................................................................
// Template strings

// templateConfigGo text template to generate FileNameConfigGo.
// NOTE the "standard header" for recognizing machine-generated files
// https://github.com/golang/go/issues/13560#issuecomment-276866852
var templateConfigGo = `
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

// templateTemplateGo text template to generate FileNameTemplateGo
var templateTemplateGo = `
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

// templateFnGo text template to generate FileNameFnGo
var templateFnGo = `
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
