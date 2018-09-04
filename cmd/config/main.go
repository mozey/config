package main

import (
	"flag"
	"log"
	"strings"
	"encoding/json"
	"github.com/mozey/logutil"
	"io/ioutil"
	"fmt"
	"os"
	"path"
	"sort"
	"bytes"
	"text/template"
	"unicode"
)

type ConfigMap map[string]string

type EnvKeys map[string]bool

// AppDir is the application root
var AppDir string
// Prefix for env vars
var Prefix = "APP"

// Flags
var Env *string
var Update *bool
var Generate *string

type ArgMap []string

func (a *ArgMap) String() string {
	return strings.Join(*a, ", ")
}
func (a *ArgMap) Set(value string) error {
	*a = append(*a, value)
	return nil
}

var Keys ArgMap
var Values ArgMap

func GetConfigPath() string {
	return path.Join(AppDir, fmt.Sprintf("config.%v.json", *Env))
}

type TemplateKey struct {
	KeyPrefix string
	KeyPrivate string
	Key string
}

type TemplateData struct {
	Prefix string
	SrcPath string
	Keys []TemplateKey
}

func ToPrivate(str string) string {
	for i, v := range str {
		return string(unicode.ToLower(v)) + str[i+1:]
	}
	return ""
}

func GenerateHelper(configKeys []string) {
	// Create template
	if Prefix != "APP" {
		configTemplate = strings.Replace(configTemplate, "APP", Prefix, -1)
	}
	t := template.Must(template.New("configTemplate").Parse(configTemplate))

	// Setup template data
	srcPath := AppDir[strings.Index(AppDir, "/src"):]
	data := TemplateData{
		Prefix: Prefix,
		SrcPath: srcPath,
	}
	for _, keyPrefix := range configKeys {
		key := strings.Replace(
			keyPrefix, fmt.Sprintf("%v_", Prefix), "", 1)
		key = strings.Replace(key, "_", " ", -1)
		key = strings.ToLower(key)
		key = strings.Replace(strings.Title(key), " ", "", -1)
		templateKey := TemplateKey{
			KeyPrefix: keyPrefix,
			KeyPrivate: ToPrivate(key),
			Key: key,
		}
		data.Keys = append(data.Keys, templateKey)
	}

	// Execute the template
	var buf bytes.Buffer
	err := t.Execute(&buf, data)
	if err != nil {
		log.Panic(err)
	}

	err = ioutil.WriteFile(
		path.Join(AppDir, *Generate, "config.go"), buf.Bytes(), 0644)
	if err != nil {
		log.Panic(err)
	}
}

func Cmd() {
	configPath := GetConfigPath()
	b, err := ioutil.ReadFile(configPath)
	if err != nil {
		logutil.Debug("Loading configPath from: ", configPath)
		log.Panic(err)
	}

	// The configPath file must have a flat key value structure
	c := ConfigMap{}
	err = json.Unmarshal(b, &c)
	if err != nil {
		log.Panic(err)
	}

	// AppDir value must be compiled with ldflags
	appDirKey := fmt.Sprintf("%v_DIR", Prefix)
	c[appDirKey] = AppDir

	// Set existing configPath Keys
	var configKeys []string
	for k := range c {
		configKeys = append(configKeys, k)
	}

	// Sort
	sort.Strings(configKeys)

	if Prefix == "" {
		log.Panicf("Prefix must not be empty")
	}

	if Generate != nil {
		// Generate helper......................................................
		GenerateHelper(configKeys)
		return

	} else if len(Keys) > 0 {
		// Set keys.............................................................

		// Validate input
		for i, key := range Keys {
			if !strings.HasPrefix(key, Prefix) {
				log.Panicf("Key must strart with prefix: %v", Prefix)
			}

			if i > len(Values)-1 {
				log.Panicf("Missing value for key: %v", key)
			}
			value := Values[i]

			// Set key value
			c[key] = value
		}

		// Update configPath
		b, _ := json.MarshalIndent(c, "", "    ")
		if *Update {
			logutil.Debug("Config updated: ", configPath)
			ioutil.WriteFile(configPath, b, 0)
		} else {
			// Print json
			fmt.Print(string(b))
		}
		return

	} else {
		// Print commands.......................................................

		// Create map of env vars starting with Prefix
		envKeys := EnvKeys{}
		for _, v := range os.Environ() {
			a := strings.Split(v, "=")
			if len(a) == 2 {
				key := a[0]
				if strings.HasPrefix(key, Prefix) {
					envKeys[a[0]] = true
				}
			}
		}

		// Print commands to set env
		for _, key := range configKeys {
			fmt.Println(fmt.Sprintf("export %v=%v", key, c[key]))
			envKeys[key] = false
		}

		// Unset env vars not listed in the config file
		for key, unset := range envKeys {
			if unset {
				fmt.Println(fmt.Sprintf("unset %v", key))
			}
		}
		return
	}
}

func main() {
	log.SetFlags(log.Lshortfile)

	Env = flag.String("env", "dev", "Specify config file to use")
	flag.Var(&Keys, "key", "Set key and print config JSON")
	flag.Var(&Values, "value", "Value for last key specified")
	Update = flag.Bool("update", false, "Update config.json")
	Generate = flag.String("gen", "", "Generate config helper at path")
	flag.Parse()

	Cmd()
}

// standard way to recognize machine-generated files
// https://github.com/golang/go/issues/13560#issuecomment-276866852
var configTemplate = `
// Code generated with https://github.com/mozey/config DO NOT EDIT

package config

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"
)

{{range .Keys}}
// {{.KeyPrefix}}
var {{.KeyPrivate}} string{{end}}


// Config fields correspond to config file keys less the prefix
type Config struct {
	{{range .Keys}}
	{{.Key}} string // {{.KeyPrefix}}{{end}}
}

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
		conf.{{.Key}} = {{.KeyPrivate}}
	}
	{{end}}
}

// SetEnv sets non-empty env vars on Config
func SetEnv(conf *Config) {
	var v string

	{{range .Keys}}
	v = os.Getenv("{{.KeyPrefix}}")
	if v != "" {
		conf.{{.Key}} = v
	}
	{{end}}
}

// LoadFile sets the env from file and returns a new instance of Config
func LoadFile(mode string) (conf *Config, err error) {
	p := fmt.Sprintf(path.Join(os.Getenv("GOPATH"),
		"{{.SrcPath}}/config.%v.json"), mode)
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
		os.Setenv(key, val)
	}
	return New(), nil
}
`