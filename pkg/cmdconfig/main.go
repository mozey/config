package cmdconfig

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/rs/zerolog/log"
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
