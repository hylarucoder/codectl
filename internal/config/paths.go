package config

import (
    "errors"
    "os"
    "path/filepath"
    "strings"
)

// Dir returns the codectl config directory under the user config base.
// On Linux, this typically resolves to $XDG_CONFIG_HOME/codectl; on macOS
// to ~/Library/Application Support/codectl; and on Windows to %AppData%/codectl.
// Falls back to HOME when UserConfigDir is unavailable.
func Dir() (string, error) {
    base, err := os.UserConfigDir()
    if err != nil || strings.TrimSpace(base) == "" {
        if home, herr := os.UserHomeDir(); herr == nil {
            base = home
        } else {
            return "", errors.New("cannot determine config directory")
        }
    }
    return filepath.Join(base, "codectl"), nil
}

