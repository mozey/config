package share

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
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
