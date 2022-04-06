package cmdconfig

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

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
	// Del deletes the specified keys
	Del bool
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
	CSV      bool
	Sep      string
	DryRun   bool
	// Base64 encode config file
	Base64 bool
	// OS overrides the compiled x-platform config
	OS string
}

// .............................................................................

type File struct {
	// Path to file
	Path string
	// Buf for new file content
	Buf *bytes.Buffer
}

type Files []File

// Print file paths and contents to buf
func (files Files) Print(buf *bytes.Buffer) {
	for _, file := range files {
		// empty file.Path implies nothing was generated
		if file.Path != "" {
			buf.WriteString("\n")
			buf.WriteString(fmt.Sprintf("// FilePath: %s", file.Path))
			buf.Write(file.Buf.Bytes())
		}
	}
}

// Save file contents to disk, and print paths to buf
func (files Files) Save(buf *bytes.Buffer) (err error) {
	// TODO Use goroutines to save files concurrently
	for _, file := range files {
		// empty file.Path implies nothing was generated
		if file.Path != "" {
			// Make sure parent dirs exist
			err := os.MkdirAll(filepath.Dir(file.Path), 0755)
			if err != nil {
				log.Info().Str("file_path", file.Path).Msg("")
				return errors.WithStack(err)
			}
			// Write the file
			err = os.WriteFile(file.Path, file.Buf.Bytes(), 0644)
			if err != nil {
				log.Info().Str("file_path", file.Path).Msg("")
				return errors.WithStack(err)
			}
			// Print file path only
			buf.WriteString(file.Path)
			buf.WriteString("\n")
		}
	}

	return nil
}

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

// ListSamples if set, otherwise list non-samples
type ListSamples bool

// GetEnvs globs all config files in APP_DIR to list possible values of env
func GetEnvs(appDir string, samples ListSamples) (envs []string, err error) {
	envs = make([]string, 0)

	// Find matching files
	fileNamePattern := "config.*.json"
	if samples {
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
		log.Error().Stack().Err(err).
			Str("config_path", configPath).Msg("")
		return configPath, c, errors.WithStack(err)
	}

	if strings.TrimSpace(string(b)) == "" {
		return configPath, c, errors.WithStack(
			fmt.Errorf("empty file %s", filepath.Base(configPath)))
	}

	// Unmarshal config.
	// The config file must have a flat key value structure
	err = json.Unmarshal(b, &c.Map)
	if err != nil {
		log.Info().Str("config_path", configPath).Msg("")
		return configPath, c, errors.WithStack(err)
	}

	RefreshKeys(c)

	// Add to cache
	configCache[configPath] = c

	return configPath, c, nil
}

// .............................................................................

// compareKeys for config files,
// buf (if not empty) contains keys that didn't match
func compareKeys(in *CmdIn) (buf *bytes.Buffer, files []File, err error) {
	buf = new(bytes.Buffer)

	_, config, err := NewConfig(in.AppDir, in.Env)
	if err != nil {
		return buf, files, err
	}
	_, compConfig, err := NewConfig(in.AppDir, in.Compare)
	if err != nil {
		return buf, files, err
	}

	unmatched := make([]string, 0, len(config.Keys)+len(compConfig.Keys))

	// Compare config keys
	for _, item := range config.Keys {
		if _, ok := compConfig.Map[item]; !ok {
			unmatched = append(unmatched, item)
		}
	}
	for _, item := range compConfig.Keys {
		if _, ok := config.Map[item]; !ok {
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

// refreshConfigByEnv replaces the given key value pairs in the specified env,
// and returns sorted JSON that can be used to replace the config file contents
func refreshConfigByEnv(
	appDir string, prefix string, env string, keys ArgMap, values ArgMap, del bool) (
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

		if del {
			// Delete the key
			_, ok := m[key]
			if ok {
				delete(m, key)
			}

		} else {
			if i > len(values)-1 {
				return configPath, b, errors.WithStack(
					fmt.Errorf("env %s missing value for key %s", env, key))
			}
			value := values[i]

			// Set value
			m[key] = value
		}

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
	var envs []string

	if in.All {
		// All config files (non-sample and sample)
		e, err := GetEnvs(in.AppDir, ListSamples(false))
		if err != nil {
			return buf, files, err
		}
		envs = append(envs, e...)
		e, err = GetEnvs(in.AppDir, ListSamples(true))
		if err != nil {
			return buf, files, err
		}
		envs = append(envs, e...)

	} else if in.Env == "*" {
		// Wildcard for non-sample config files
		envs, err = GetEnvs(in.AppDir, ListSamples(false))
		if err != nil {
			return buf, files, err
		}

	} else if in.Env == "sample.*" {
		// Wildcard for sample config files
		envs, err = GetEnvs(in.AppDir, ListSamples(true))
		if err != nil {
			return buf, files, err
		}

	} else {
		// Only the config file as per the env flag
		envs = append(envs, in.Env)
	}

	// Refresh config for the listed envs
	files = make([]File, len(envs))
	for i, env := range envs {
		var configPath string
		configPath, b, err = refreshConfigByEnv(
			in.AppDir, in.Prefix, env, in.Keys, in.Values, in.Del)
		if err != nil {
			return buf, files, err
		}
		files[i] = File{
			Path: configPath,
			Buf:  bytes.NewBuffer(b),
		}
	}

	return buf, files, nil
}

// .............................................................................

type EnvKeys map[string]bool

func setEnv(in *CmdIn) (buf *bytes.Buffer, files []File, err error) {
	_, config, err := NewConfig(in.AppDir, in.Env)
	if err != nil {
		return buf, files, err
	}

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

	// Default format is determined at compile time
	exportFormat := ExportFormat
	unsetFormat := UnsetFormat

	// Override default format by specifying os flag
	if in.OS == "windows" {
		exportFormat = WindowsExportFormat
		unsetFormat = WindowsUnsetFormat
	} else if in.OS == "linux" || in.OS == "darwin" {
		exportFormat = OtherExportFormat
		unsetFormat = OtherUnsetFormat
	}

	// Commands to set env
	for _, key := range config.Keys {
		buf.WriteString(fmt.Sprintf(exportFormat, key, config.Map[key]))
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
			buf.WriteString(fmt.Sprintf(unsetFormat, key))
			buf.WriteString("\n")
		}
	}

	return buf, files, nil
}

// .............................................................................

func generateCSV(in *CmdIn) (buf *bytes.Buffer, files []File, err error) {
	buf = new(bytes.Buffer)

	_, config, err := NewConfig(in.AppDir, in.Env)
	if err != nil {
		return buf, files, err
	}

	a := make([]string, len(config.Keys))
	for i, key := range config.Keys {
		value := config.Map[key]
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

	_, config, err := NewConfig(in.AppDir, in.Env)
	if err != nil {
		return buf, files, err
	}

	b, err := json.Marshal(config.Map)
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

	_, config, err := NewConfig(in.AppDir, in.Env)
	if err != nil {
		return buf, files, err
	}

	if value, ok := config.Map[key]; ok {
		buf.WriteString(value)
		return buf, files, nil
	}

	return buf, files, errors.WithStack(
		fmt.Errorf("missing value for key %v", key))
}
