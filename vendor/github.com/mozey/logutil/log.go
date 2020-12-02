package logutil

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"io"
	"os"
	"runtime"
	"time"
)

// SetDefaults as recommended by this package
func SetDefaults() {
	zerolog.TimeFieldFormat = time.RFC3339
	zerolog.TimestampFieldName = "created"
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	zerolog.ErrorFieldName = "message"
	zerolog.ErrorStackMarshaler = MarshalStack
}

// SetupLogger sets up logging using zerolog.
//
// Wrap new errors with WithStack
//		errors.WithStack(fmt.Errorf("foo"))
//
// Errors returned by build-in or third party packages
// should be wrapped using `errors.WithStack`.
// Avoid excessive use of `errors.Wrap`,
// it's not as useful as a stack trace,
// and makes the error message harder to read.
//
// Call `.WithStack` on the boundaries of your project.
// Then don't call it again internally to the project.
// The stack trace must take you to the line where
// your project is interfacing with the vendor code
//
// Additional writer may be specified, for example to log to a file
//	f, err := os.OpenFile(pathToFile, os.O_WRONLY|os.O_CREATE, 0644)
//	logutil.SetupLogger(true, f)
// See https://github.com/rs/zerolog#multiple-log-output
//
func SetupLogger(consoleWriter bool, w ...io.Writer) {
	SetDefaults()

	writers := make([]io.Writer, 0, len(w)+1)

	if consoleWriter {
		// Dev
		zerolog.SetGlobalLevel(zerolog.DebugLevel)

		// Windows cmd.exe doesn't support escape sequences for colors
		noColor := false
		if runtime.GOOS == "windows" {
			noColor = true
		}

		writer := ConsoleWriter{
			Out:           os.Stdout,
			NoColor:       noColor,
			TimeFormat:    "2006-01-02 15:04:05",
			MarshalIndent: true,
		}
		writers = append(writers, writer)

	} else {
		// Prod
		writer := log.With().Caller().Logger()
		writers = append(writers, writer)
	}

	// Log to additional writers, e.g. file
	if len(w) > 0 {
		writers = append(writers, w...)
	}

	multi := zerolog.MultiLevelWriter(writers...)
	log.Logger = zerolog.New(multi).With().Timestamp().Caller().Logger()
}

func PanicHandler() {
	if r := recover(); r != nil {
		err := fmt.Errorf("%s", r)
		// Use zerolog to print stack trace
		// https://github.com/rs/zerolog/pull/35
		err = errors.Wrap(err, "recovered panic")
		log.Error().Stack().Err(err).Msg("")
	}
}

// LogToFile only
func LogToFile(f *os.File) {
	SetDefaults()
	log.Logger = zerolog.New(f)
}
