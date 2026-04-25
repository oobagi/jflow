package state

import (
	"os"
	"path/filepath"
)

// StateDir returns ~/.jflow/state, creating it if missing.
func StateDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".jflow", "state")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return dir, nil
}

// LogsDir returns ~/.jflow/state/logs, creating it if missing.
func LogsDir() (string, error) {
	s, err := StateDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(s, "logs")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return dir, nil
}
