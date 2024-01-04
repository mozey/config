package cmdconfig

import (
	"bytes"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"
	"unicode"

	"github.com/pkg/errors"
)

const TokenTemplateKey = "_TEMPLATE_"

const TokenExtendedConfigKey = "_X_"

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

func NewGenerateData(in *CmdIn) (data *GenerateData, err error) {
	// Init
	data = &GenerateData{
		Prefix: in.Prefix,
		AppDir: in.AppDir,
	}

	_, config, err := newConf(in.AppDir, in.Env)
	if err != nil {
		return data, err
	}

	// APP_DIR is usually not set in the config.json file
	keys := make([]string, len(config.Keys))
	copy(keys, config.Keys)
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
		if strings.Contains(keyWithPrefix, TokenTemplateKey) {
			templateKeys = append(templateKeys, generateKey)
		}
	}

	// Template keys are use to generate template.go
	for _, generateKey := range templateKeys {
		templateKey := TemplateKey{
			GenerateKey: generateKey,
		}
		params := GetTemplateParams(config.Map[generateKey.KeyPrefix])
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

	return data, nil
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
func generateHelpers(in *CmdIn) (files []File, err error) {
	// Generate data for executing template
	data, err := NewGenerateData(in)
	if err != nil {
		return files, err
	}

	// NOTE buf is usually filled with content to be written to stdout.
	// For the generate flag the contents of buf depends on the dry run flag,
	// and that is checked elsewhere

	// files contains file paths and generated code,
	// depending on the dry run flag it may be written
	// to stdout or the file system
	files = make([]File, 3)

	filePath, buf, err := executeTemplate(in, FileNameConfigGo, data)
	if err != nil {
		return files, err
	}
	files[0] = File{
		Path: filePath,
		Buf:  bytes.NewBuffer(buf.Bytes()),
	}

	if len(data.TemplateKeys) > 0 {
		filePath, buf, err = executeTemplate(in, FileNameTemplateGo, data)
		if err != nil {
			return files, err
		}
		files[1] = File{
			Path: filePath,
			Buf:  bytes.NewBuffer(buf.Bytes()),
		}
	} else {
		files[1] = File{
			Path: "",
			Buf:  bytes.NewBuffer([]byte("")),
		}
	}

	filePath, buf, err = executeTemplate(in, FileNameFnGo, data)
	if err != nil {
		return files, err
	}
	files[2] = File{
		Path: filePath,
		Buf:  bytes.NewBuffer(buf.Bytes()),
	}

	return files, nil
}
