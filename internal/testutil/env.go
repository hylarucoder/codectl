package testutil

import "testing"
import "os"

// WithEnv sets env var to val for the duration of the test scope.
// Returns a cleanup func to restore previous value.
func WithEnv(t *testing.T, key, val string) func() {
    t.Helper()
    old, had := os.LookupEnv(key)
    if val == "" {
        _ = os.Unsetenv(key)
    } else {
        _ = os.Setenv(key, val)
    }
    return func() {
        if had {
            _ = os.Setenv(key, old)
        } else {
            _ = os.Unsetenv(key)
        }
    }
}

