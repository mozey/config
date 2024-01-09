package cmdconfig

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v2"
)

// .............................................................................

// conf file attributes.
// Note, this is not the same struct as the generated config.conf,
// e.g. pkg/cmdconfig/testdata/config.go
// The latter has properties for each config attribute in a project,
// whereas this type is generic
type conf struct {
	// Map of key to value
	Map map[string]string
	// Keys sorted
	Keys []string
}

func (c *conf) refreshKeys() {
	c.Keys = nil
	// Set config keys
	for k := range c.Map {
		c.Keys = append(c.Keys, k)
	}
	// Sort keys
	sort.Strings(c.Keys)
}

// extend config with another config, keys must be unique.
// Remember to call refreshKeys afterwards
func (c *conf) extend(ext *conf) error {
	for k, v := range ext.Map {
		_, dup := c.Map[k]
		if dup {
			return ErrDuplicateKey(k)
		}
		c.Map[k] = v
	}
	return nil
}

// .............................................................................

// CmdIn for use with command functions
type CmdIn struct {
	// version is the build version
	version string
	// AppDir is the application root
	AppDir string
	// Prefix for env vars
	Prefix string
	// PrintVersion for printing the build version
	PrintVersion bool
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
	// Override config file format
	Format string
	// Extend config
	Extend ArgMap
	// Merge with parent config
	Merge bool
}

type CmdInParams struct {
	// Version to print with the version flag
	Version string
}

// NewCmdIn constructor for CmdIn
func NewCmdIn(params CmdInParams) *CmdIn {
	return &CmdIn{
		version: params.Version,
	}
}

// Valid returns true if the command input is valid.
// It may also set default values
func (in *CmdIn) Valid() error {
	// AppDir is required
	appDirKey := fmt.Sprintf("%sDIR", in.Prefix)
	appDir := os.Getenv(appDirKey)
	if appDir == "" {
		return errors.Errorf("%v env not set%s", appDirKey, "\n")
	}
	in.AppDir = appDir

	// Prefix must end with underscore
	prefix := in.Prefix
	if prefix[len(prefix)-1:] != "_" {
		in.Prefix = fmt.Sprintf("%s_", prefix)
	}

	return nil
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
	// ExitCode can be non-zero if the err returned is nil,
	// that means the program did not have any internal error,
	// but the command "failed", i.e. non-zero exit code
	ExitCode int
	// Buf of cmd output
	Buf *bytes.Buffer
	// Files to write if in.DryRun is not set
	Files Files
}

// .............................................................................

// listSamples if set, otherwise list non-samples
type listSamples bool

// getEnvs globs all config files in APP_DIR to list possible values of env
func getEnvs(appDir string, samples listSamples) (envs []string, err error) {
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

	// Regexp to submatch env from file name.
	// Env must start with a word character, and may contain hyphens
	s := "config\\.(\\w+[\\w\\-]*)\\.json"
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

const FileTypeEnv = ".env"   // e.g. .env
const FileTypeJSON = ".json" // e.g. config.json
const FileTypeYAML = ".yaml" // e.g. config.yaml

// getConfigFilePath returns the path to a config file.
// It can also be used to return paths to sample config file by prefixing env,
// for example, to get the path to "sample.config.dev.json" pass env="sample.dev"
func getConfigFilePath(appDir, env, fileType string) (string, error) {
	if _, err := os.Stat(appDir); err != nil {
		if os.IsNotExist(err) {
			return "", errors.Errorf("app dir does not exist %v", appDir)
		} else {
			return "", errors.WithStack(err)
		}
	}

	// Strip SamplePrefix from env
	sample := "" // e.g. config.dev.json
	if strings.Contains(env, SamplePrefix) {
		sample = SamplePrefix
		env = strings.Replace(env, SamplePrefix, "", 1)
	}

	// If env is not empty, add dot separator.
	if fileType != FileTypeEnv {
		if strings.TrimSpace(env) != "" {
			env = fmt.Sprintf(".%s", env)
		}
	}

	// Format for FileTypeEnv is slightly different,
	// it does not contain the word "config" (by popular convention)
	fileNameFormat := "%vconfig%v%v" // e.g. sample.config.dev.json
	if fileType == FileTypeEnv {
		fileNameFormat = "%v%v%v" // e.g. sample.dev.env
	}

	return filepath.Join(
		appDir, fmt.Sprintf(fileNameFormat, sample, env, fileType)), nil
}

// getConfigFilePaths defines the load precedence
func getConfigFilePaths(appDir, env string) (paths []string, err error) {
	paths = []string{}

	for _, fileType := range []string{
		// Load precedence
		FileTypeJSON,
		FileTypeEnv,
		FileTypeYAML,
	} {
		configPath, err := getConfigFilePath(appDir, env, fileType)
		if err != nil {
			return paths, err
		}
		paths = append(paths, configPath)

		// For the dev config file, the env is optional, i.e.
		// "config.dev.json" or "config.json" are both valid dev config files
		configPath, err = getConfigFilePath(appDir, "", fileType)
		if err != nil {
			return paths, err
		}
		paths = append(paths, configPath)
	}

	return paths, nil
}

func ReadConfigFile(appDir, env string) (configPath string, b []byte, err error) {
	found := false
	paths, err := getConfigFilePaths(appDir, env)
	if err != nil {
		return configPath, b, err
	}
	// Don't change scope of configPath variable!
	for _, configPath = range paths {
		_, err := os.Stat(configPath)
		if err != nil {
			if os.IsNotExist(err) {
				// log.Debug().Str("config_path", configPath).Msg("Not found")
				continue
			} else {
				return configPath, b, errors.WithStack(err)
			}
		}

		// Config file exists, try to read it
		b, err = os.ReadFile(configPath)
		if err != nil {
			log.Error().Stack().Err(err).
				Str("config_path", configPath).Msg("")
			return configPath, b, errors.WithStack(err)
		}
		found = true
		break
	}

	if !found {
		return configPath, b, errors.Errorf(
			"config file not found for env %s", env)
	}
	// log.Debug().Str("config_path", configPath).Msg("Found")

	if strings.TrimSpace(string(b)) == "" {
		return configPath, b, errors.Errorf(
			"empty file %s", filepath.Base(configPath))
	}
	return configPath, b, nil
}

// loadConf loads config from a file
func loadConf(appDir string, env string) (
	configPath string, c *conf, err error) {

	// New config
	c = &conf{}

	configPath, b, err := ReadConfigFile(appDir, env)
	if err != nil {
		return configPath, c, err
	}

	// Unmarshal config.
	// The config file must have a flat key value structure
	fileType := filepath.Ext(configPath)
	var UnmarshalErr error
	if fileType == FileTypeEnv {
		c.Map, UnmarshalErr = UnmarshalENV(b)
	} else if fileType == FileTypeJSON {
		UnmarshalErr = json.Unmarshal(b, &c.Map)
	} else if fileType == FileTypeYAML {
		UnmarshalErr = yaml.Unmarshal(b, &c.Map)
	}
	if UnmarshalErr != nil {
		log.Info().Str("config_path", configPath).Msg("")
		return configPath, c, errors.WithStack(UnmarshalErr)
	}

	c.refreshKeys()

	return configPath, c, nil
}

type confParams struct {
	appDir string
	env    string
	extend []string
	merge  bool
}

// newConf constructor for conf
func newConf(params confParams) (
	configPaths []string, c *conf, err error) {

	if len(params.extend) > 0 {
		// Extend config
		return newExtendedConf(params)

	} else if params.merge {
		// Merge with parent config
		return newMergedConf(params)
	}

	// Default
	return newSingleConf(params.appDir, params.env)
}

// newSingleConf reads a config file and sets the key map
func newSingleConf(appDir string, env string) (configPaths []string, c *conf, err error) {
	configPath, c, err := loadConf(appDir, env)
	if err != nil {
		return configPaths, c, err
	}
	configPaths = append(configPaths, configPath)

	return configPaths, c, nil
}

// newExtendedConf reads config from multiple files.
// The main config file in the APP_DIR is extended
// with config files from extensions in sub dirs
// https://github.com/mozey/config/issues/47
func newExtendedConf(params confParams) (
	configPaths []string, c *conf, err error) {

	if params.merge {
		// TODO Support both extend and merge?
		// Note that APP_DIR is always set to the current working directory.
		// If both extend and merge are set, then that implies
		// the project structure looks like this: parent/current/extension,
		// that is three levels of config files.
		// Don't implement this unless there is a specific use case
		return configPaths, c, ErrNotImplemented
	}

	// Main config
	configPaths, c, err = newSingleConf(params.appDir, params.env)
	if err != nil {
		return configPaths, c, err
	}

	// Try to load the extension config
	for _, extDir := range params.extend {
		configPath, extConf, err := loadConf(
			filepath.Join(params.appDir, extDir), params.env)
		if err != nil {
			return configPaths, c, err
		}
		configPaths = append(configPaths, configPath)
		// Extend main config
		err = c.extend(extConf)
		if err != nil {
			return configPaths, c, err
		}
	}

	c.refreshKeys()

	return configPaths, c, nil
}

// newMergedConf merges an extension with a parent config file
func newMergedConf(params confParams) (
	configPaths []string, c *conf, err error) {

	// TODO Search for parent relative to appDir
	parentDir := ""

	// Parent config
	configPath, c, err := loadConf(parentDir, params.env)
	if err != nil {
		return configPaths, c, err
	}
	configPaths = append(configPaths, configPath)

	// Extended config
	configPath, extConf, err := loadConf(params.appDir, params.env)
	if err != nil {
		return configPaths, c, err
	}
	configPaths = append(configPaths, configPath)

	// Merge with parent config
	err = c.extend(extConf)
	if err != nil {
		return configPaths, c, err
	}

	c.refreshKeys()

	return configPaths, c, nil
}

// .............................................................................

// compareKeys for config files,
// buf (if not empty) contains keys that didn't match
func compareKeys(in *CmdIn) (buf *bytes.Buffer, files []File, err error) {
	buf = new(bytes.Buffer)

	_, config, err := newConf(confParams{
		appDir: in.AppDir,
		env:    in.Env,
		extend: in.Extend,
		merge:  in.Merge,
	})
	if err != nil {
		return buf, files, err
	}
	_, compConfig, err := newConf(confParams{
		appDir: in.AppDir,
		env:    in.Compare,
		extend: in.Extend,
		merge:  in.Merge,
	})
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
// and returns sorted bytes that can be used to update the config file
func refreshConfigByEnv(
	appDir string, prefix string, env string, keys ArgMap, values ArgMap,
	del bool, format string) (
	configPaths []string, b []byte, err error) {

	// Read config for the given env from file
	configPaths, conf, err := newSingleConf(appDir, env)
	if err != nil {
		return configPaths, b, err
	}

	// Setup existing key value pairs
	m := make(map[string]string)
	for _, key := range conf.Keys {
		m[key] = conf.Map[key]
	}

	// Validate input
	for i, key := range keys {
		if !strings.HasPrefix(key, prefix) {
			return configPaths, b, errors.Errorf(
				"key for env %s must start with prefix %s", env, prefix)
		}

		if del {
			// Delete the key
			_, ok := m[key]
			if ok {
				delete(m, key)
			}

		} else {
			if i > len(values)-1 {
				return configPaths, b, errors.Errorf(
					"env %s missing value for key %s", env, key)
			}
			value := values[i]

			// Set value
			m[key] = value
		}

		conf.refreshKeys()
	}

	// Marshal config
	if len(configPaths) == 0 {
		return configPaths, b, errors.Errorf("empty config path")
	}
	fileType := filepath.Ext(configPaths[0])
	var MarshalErr error
	dotFormat := fmt.Sprintf(".%s", format)
	if dotFormat == FileTypeEnv ||
		dotFormat == FileTypeJSON ||
		dotFormat == FileTypeYAML {
		//	Override config file format
		fileType = dotFormat
		configPaths[0], err = getConfigFilePath(appDir, env, dotFormat)
		if err != nil {
			return configPaths, b, err
		}
	}
	if fileType == FileTypeEnv {
		b, MarshalErr = MarshalENV(m)
	} else if fileType == FileTypeJSON {
		b, MarshalErr = json.MarshalIndent(m, "", "    ")
	} else if fileType == FileTypeYAML {
		b, MarshalErr = yaml.Marshal(m)
	}
	if MarshalErr != nil {
		return configPaths, b, errors.WithStack(MarshalErr)
	}

	return configPaths, b, nil
}

func updateConfig(in *CmdIn) (buf *bytes.Buffer, files []File, err error) {
	buf = new(bytes.Buffer)
	var b []byte
	var envs []string

	if in.All {
		// All config files (non-sample and sample)
		e, err := getEnvs(in.AppDir, listSamples(false))
		if err != nil {
			return buf, files, err
		}
		envs = append(envs, e...)
		e, err = getEnvs(in.AppDir, listSamples(true))
		if err != nil {
			return buf, files, err
		}
		envs = append(envs, e...)

	} else if in.Env == "*" {
		// Wildcard for non-sample config files
		envs, err = getEnvs(in.AppDir, listSamples(false))
		if err != nil {
			return buf, files, err
		}

	} else if in.Env == "sample.*" {
		// Wildcard for sample config files
		envs, err = getEnvs(in.AppDir, listSamples(true))
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
		var configPaths []string
		configPaths, b, err = refreshConfigByEnv(
			in.AppDir, in.Prefix, env, in.Keys, in.Values, in.Del, in.Format)
		if err != nil {
			return buf, files, err
		}
		if len(configPaths) == 0 {
			return buf, files, errors.Errorf("empty config path")
		}
		files[i] = File{
			Path: configPaths[0],
			Buf:  bytes.NewBuffer(b),
		}
	}

	return buf, files, nil
}

// .............................................................................

type envKeys map[string]bool

// setEnv commands to be executed in the shell
func setEnv(in *CmdIn) (buf *bytes.Buffer, files []File, err error) {
	_, config, err := newSingleConf(in.AppDir, in.Env)
	if err != nil {
		return buf, files, err
	}

	// Create map of env vars starting with Prefix
	envKeys := envKeys{}
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

	_, config, err := newConf(confParams{
		appDir: in.AppDir,
		env:    in.Env,
		extend: in.Extend,
		merge:  in.Merge,
	})
	if err != nil {
		return buf, files, err
	}

	a := make([]string, len(config.Keys))
	for i, key := range config.Keys {
		value := config.Map[key]
		if strings.Contains(value, "\n") {
			return buf, files, errors.Errorf("values must not contain newlines")
		}
		if strings.Contains(value, ",") {
			return buf, files, errors.Errorf("values must not contain commas")
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

	_, config, err := newConf(confParams{
		appDir: in.AppDir,
		env:    in.Env,
		extend: in.Extend,
		merge:  in.Merge,
	})
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

	_, config, err := newConf(confParams{
		appDir: in.AppDir,
		env:    in.Env,
		extend: in.Extend,
		merge:  in.Merge,
	})
	if err != nil {
		return buf, files, err
	}

	if value, ok := config.Map[key]; ok {
		buf.WriteString(value)
		return buf, files, nil
	}

	return buf, files, errors.Errorf("missing value for key %v", key)
}
