package cmdconfig

import (
	"flag"
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

const (
	FlagAll      = "all"
	FlagBase64   = "base64"
	FlagCompare  = "compare"
	FlagCSV      = "csv"
	FlagDel      = "del"
	FlagDryRun   = "dry-run"
	FlagEnv      = "env"
	FlagGenerate = "generate"
	FlagGet      = "get"
	FlagKey      = "key"
	FlagPrefix   = "prefix"
	FlagSep      = "sep"
	FlagValue    = "value"
	FlagVersion  = "version"
	FlagOS       = "os"
	FlagFormat   = "format"
)

// ParseFlags before calling Cmd
func ParseFlags(version string) *CmdIn {
	in := NewCmdIn(CmdInParams{Version: version})

	// Flags
	flag.BoolVar(&in.PrintVersion,
		FlagVersion, false, "Print build version")
	flag.StringVar(&in.Prefix,
		FlagPrefix, "APP_", "Config key prefix")
	flag.StringVar(&in.Env,
		FlagEnv, "dev",
		"Config file to use, also supports wildcards \"*\" and \"sample.*\"")
	flag.BoolVar(&in.All,
		FlagAll, false, "Apply to all config files and samples")
	flag.BoolVar(&in.Del,
		FlagDel, false, "Delete the specified keys")
	// Default must be empty
	flag.StringVar(&in.Compare,
		FlagCompare, "", "Compare config file keys")
	in.Keys = ArgMap{}
	flag.Var(&in.Keys,
		FlagKey, "Set key and print config JSON")
	in.Values = ArgMap{}
	flag.Var(&in.Values,
		FlagValue, "Value for last key specified")
	// Default must be empty
	flag.StringVar(&in.PrintValue,
		FlagGet, "", "Print value for given key")
	// Default must be empty
	flag.StringVar(&in.Generate,
		FlagGenerate, "", "Generate config helper at path")
	flag.BoolVar(&in.CSV,
		FlagCSV, false, "Print env as a list of key=value")
	flag.StringVar(&in.Sep,
		FlagSep, ",", "Separator for use with csv flag")
	flag.BoolVar(&in.DryRun,
		FlagDryRun, false, "Don't write files, just print result")
	flag.BoolVar(&in.Base64,
		FlagBase64, false, "Encode config file as base64 string")
	flag.StringVar(&in.OS,
		FlagOS, "other",
		"Override compiled x-platform config")
	flag.StringVar(&in.Format,
		FlagFormat, "", "Override config file format")

	flag.Parse()

	return in
}

// Main function for cmd/configu.
// The configu command can be customized by copying the code below.
// Try not to change the default behaviour, e.g.
// custom flags must only add functionality
func Main(version string) {
	// Parse and validate flags
	in := ParseFlags(version)
	err := in.Valid()
	if err != nil {
		log.Error().Stack().Err(err).Msg("")
		os.Exit(1)
	}

	// Insert your custom code here...

	// Run cmd
	out, err := Cmd(in)
	if err != nil {
		log.Error().Stack().Err(err).Msg("")
		os.Exit(1)
	}

	// Process cmd results
	exitCode, err := in.Process(out)
	if err != nil {
		log.Error().Stack().Err(err).Msg("")
	}
	os.Exit(exitCode)
}
