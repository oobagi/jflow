// Package config holds user preferences persisted at ~/.jflow/config.json.
// Keep the struct small — fields are added as features need them.
package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/oobagi/jflow/internal/state"
)

// Preferences are the persisted user preferences. Zero values mean "fall
// back to a sensible default at use time".
type Preferences struct {
	// DefaultDir is the directory used by general (workspace-less) sessions.
	// When empty, callers fall back to the current working directory.
	DefaultDir string `json:"default_dir,omitempty"`
}

// DefaultPath returns ~/.jflow/config.json.
func DefaultPath() (string, error) {
	dir, err := state.JflowDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.json"), nil
}

// Load reads preferences from path. A missing file returns a zero-value
// Preferences (no error) so first-run callers don't need to special-case it.
func Load(path string) (Preferences, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Preferences{}, nil
		}
		return Preferences{}, fmt.Errorf("read %s: %w", path, err)
	}
	if len(data) == 0 {
		return Preferences{}, nil
	}
	var p Preferences
	if err := json.Unmarshal(data, &p); err != nil {
		return Preferences{}, fmt.Errorf("unmarshal %s: %w", path, err)
	}
	return p, nil
}

// LoadDefault is shorthand for Load(DefaultPath()). Errors fall through; the
// returned struct is zero-valued.
func LoadDefault() (Preferences, error) {
	p, err := DefaultPath()
	if err != nil {
		return Preferences{}, err
	}
	return Load(p)
}

// Save writes preferences atomically (tmp + rename).
func Save(path string, p Preferences) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", filepath.Dir(path), err)
	}
	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal preferences: %w", err)
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", tmp, err)
	}
	if err := os.Rename(tmp, path); err != nil {
		return fmt.Errorf("rename %s: %w", tmp, err)
	}
	return nil
}

// SaveDefault writes preferences to ~/.jflow/config.json.
func SaveDefault(p Preferences) error {
	path, err := DefaultPath()
	if err != nil {
		return err
	}
	return Save(path, p)
}

// ResolveDefaultDir returns the configured default directory (tilde-expanded
// against $HOME) or empty if no preference is set. Validation that the path
// exists is left to the caller.
func (p Preferences) ResolveDefaultDir() string {
	d := p.DefaultDir
	if d == "" {
		return ""
	}
	if d == "~" || (len(d) > 1 && d[0] == '~' && (d[1] == '/' || d[1] == os.PathSeparator)) {
		if home, err := os.UserHomeDir(); err == nil {
			if d == "~" {
				return home
			}
			return filepath.Join(home, d[2:])
		}
	}
	abs, err := filepath.Abs(d)
	if err != nil {
		return d
	}
	return abs
}
