package cmdconfig

import (
	"bytes"
	"fmt"
	"os"
)

const (
	CmdBase64       = "base64"
	CmdCompare      = "compare"
	CmdCSV          = "csv"
	CmdGenerate     = "generate"
	CmdGet          = "get"
	CmdSetEnv       = "set-env"
	CmdUpdateConfig = "update-config"
)

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
				out.Files.Print(out.Buf)
			}
		} else {
			// Create or update the files
			out.Files.Save(out.Buf)
		}
		fmt.Println(out.Buf.String())
		os.Exit(out.ExitCode)

	case CmdGenerate:
		// .....................................................................
		if out.Buf == nil {
			out.Buf = new(bytes.Buffer)
		}
		if in.DryRun {
			// Print file paths and generated text
			out.Files.Print(out.Buf)
		} else {
			// Create or update the files
			out.Files.Save(out.Buf)
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
