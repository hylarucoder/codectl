package system

import (
    "os"

    clog "github.com/charmbracelet/log"
)

// Logger is the shared application logger for CLI output.
// It prints to stderr with timestamps enabled for better UX.
var Logger = clog.NewWithOptions(os.Stderr, clog.Options{
    ReportTimestamp: true,
})

