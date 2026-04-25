package state

import (
	"os"
	"path/filepath"
)

// JflowDir returns ~/.jflow, creating it if missing. Used for top-level
// files like config.json that don't belong inside the state subdir.
func JflowDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".jflow")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return dir, nil
}

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
