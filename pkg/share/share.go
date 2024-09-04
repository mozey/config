package share

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

const EnvDev = "dev"

const FileTypeENV = ".env"   // e.g. .env
const FileTypeSH = ".sh"     // e.g. .env.prod.sh
const FileTypeJSON = ".json" // e.g. config.json
const FileTypeYAML = ".yaml" // e.g. config.yaml

func LoadPrecedence() []string {
	return []string{
		FileTypeENV,
		FileTypeSH,
		FileTypeJSON,
		FileTypeYAML,
	}
}

const Sample = "sample"

func SamplePrefix() string {
	return fmt.Sprintf("%s.", Sample)
}

// GetConfigFilePath returns the path to a config file.
// It can also be used to return paths to sample config file by prefixing env,
// for example, to get the path to "sample.config.dev.json" pass env="sample.dev"
func GetConfigFilePath(appDir, env, fileType string) (string, error) {
	if _, err := os.Stat(appDir); err != nil {
		if os.IsNotExist(err) {
			return "", errors.Errorf("app dir does not exist %v", appDir)
		} else {
			return "", errors.WithStack(err)
		}
	}

	// Strip sample prefix from env
	env = strings.TrimSpace(env)
	sample := ""
	samplePrefix := SamplePrefix()
	if strings.Contains(env, samplePrefix) {
		sample = Sample
		env = strings.Replace(env, samplePrefix, "", 1)
	} else {
		samplePrefix = ""
	}

	// Text editors usually do syntax highlighting for ".env" files
	if fileType == FileTypeENV && sample == "" && env == "" {
		return filepath.Join(appDir, ".env"), nil
	}

	// If env is not empty, add dot separator.
	if env != "" {
		env = fmt.Sprintf(".%s", env)
	}

	// For environements other than dev, or sample files,
	// the filename must end with ".sh"
	if fileType == FileTypeSH {
		// E.g. .env.prod.sh or sample.env.prod.sh
		fileNameFormat := "%v.env%v%v"
		return filepath.Join(
			appDir, fmt.Sprintf(fileNameFormat, sample, env, fileType)), nil
	}

	// E.g. config.dev.json or sample.config.dev.json
	fileNameFormat := "%vconfig%v%v"
	return filepath.Join(
		appDir, fmt.Sprintf(fileNameFormat, samplePrefix, env, fileType)), nil
}

// GetConfigFilePaths returns paths config files might be loaded from
func GetConfigFilePaths(appDir, env string) (paths []string, err error) {
	paths = []string{}

	for _, fileType := range LoadPrecedence() {
		if fileType != FileTypeENV {
			configPath, err := GetConfigFilePath(appDir, env, fileType)
			if err != nil {
				return paths, err
			}
			paths = append(paths, configPath)
		}

		if env == EnvDev {
			// For the dev config file, the env is optional, i.e.
			// "config.dev.json" or "config.json" are both valid dev config files
			configPath, err := GetConfigFilePath(appDir, "", fileType)
			if err != nil {
				return paths, err
			}
			paths = append(paths, configPath)
		}
	}

	return paths, nil
}

// UnmarshalENV .env file bytes to key value map.
// Syntax rules as per this comment
// https://github.com/mozey/config/issues/24#issue-1091975787
func UnmarshalENV(b []byte) (m map[string]string, err error) {
	m = make(map[string]string)

	// Using multi-line mode regex
	// https://stackoverflow.com/a/62996933/639133
	expr := "(?m)^\\s*([_a-zA-Z0-9]+)\\s*=\\s*(.+)\\s*$"
	r, _ := regexp.Compile(expr)
	lines := r.FindAllString(string(b), -1)

	for _, line := range lines {
		key, value, found := strings.Cut(line, "=")
		if !found {
			return m, errors.Errorf("regexp error %s", line)
		}

		// Trim surrounding white space
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)

		// Remove surrounding, quotes inside the value is kept
		value = strings.TrimPrefix(value, "\"")
		value = strings.TrimSuffix(value, "\"")

		m[key] = value
	}

	return m, nil
}

func UnmarshalConfig(configPath string, b []byte) (
	configMap map[string]string, err error) {

	// Unmarshal config.
	// The config file must have a flat key value structure
	fileType := filepath.Ext(configPath)
	if fileType == FileTypeENV || fileType == FileTypeSH {
		configMap, err = UnmarshalENV(b)
	} else if fileType == FileTypeJSON {
		err = json.Unmarshal(b, &configMap)
	} else if fileType == FileTypeYAML {
		err = yaml.Unmarshal(b, &configMap)
	}
	if err != nil {
		return configMap, errors.WithStack(err)
	}

	return configMap, nil
}
