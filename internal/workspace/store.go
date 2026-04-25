package workspace

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/oobagi/jflow/internal/state"
)

// fileVersion is the on-disk schema version for workspaces.json.
const fileVersion = 1

// fileShape is the JSON envelope persisted to disk. Workspaces are stored as
// an ordered list (not a map) so listing is stable.
type fileShape struct {
	Version    int         `json:"version"`
	Workspaces []Workspace `json:"workspaces"`
}

// Store wraps ~/.jflow/state/workspaces.json. All access goes through a
// single mutex; saves are atomic via temp file + rename.
type Store struct {
	mu         sync.Mutex
	path       string
	workspaces []Workspace
}

// ErrNotFound is returned when a workspace lookup misses.
var ErrNotFound = errors.New("workspace not found")

// ErrAmbiguous is returned when a query (e.g. an ID prefix) matches more than
// one workspace.
var ErrAmbiguous = errors.New("ambiguous workspace selector")

// ErrExists is returned when adding a workspace whose cwd already has one.
var ErrExists = errors.New("workspace already exists for cwd")

// DefaultPath returns ~/.jflow/state/workspaces.json.
func DefaultPath() (string, error) {
	dir, err := state.StateDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "workspaces.json"), nil
}

// Open loads (or creates) the store at the given path. A missing file is not
// an error — it just yields an empty store.
func Open(path string) (*Store, error) {
	s := &Store{path: path}
	if err := s.load(); err != nil {
		return nil, err
	}
	return s, nil
}

// OpenDefault opens the store at ~/.jflow/state/workspaces.json.
func OpenDefault() (*Store, error) {
	p, err := DefaultPath()
	if err != nil {
		return nil, err
	}
	return Open(p)
}

func (s *Store) load() error {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			s.workspaces = nil
			return nil
		}
		return fmt.Errorf("read %s: %w", s.path, err)
	}
	if len(data) == 0 {
		s.workspaces = nil
		return nil
	}
	var f fileShape
	if err := json.Unmarshal(data, &f); err != nil {
		return fmt.Errorf("unmarshal %s: %w", s.path, err)
	}
	s.workspaces = f.Workspaces
	return nil
}

// save writes the current workspace list atomically. Caller must hold s.mu.
func (s *Store) save() error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", filepath.Dir(s.path), err)
	}
	f := fileShape{Version: fileVersion, Workspaces: s.workspaces}
	if f.Workspaces == nil {
		f.Workspaces = []Workspace{}
	}
	data, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal workspaces: %w", err)
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", tmp, err)
	}
	if err := os.Rename(tmp, s.path); err != nil {
		return fmt.Errorf("rename %s: %w", tmp, err)
	}
	return nil
}

// List returns a copy of all workspaces in insertion order.
func (s *Store) List() []Workspace {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Workspace, len(s.workspaces))
	copy(out, s.workspaces)
	return out
}

// Get returns a copy of the workspace with the given ID.
func (s *Store) Get(id string) (Workspace, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, w := range s.workspaces {
		if w.ID == id {
			return w, nil
		}
	}
	return Workspace{}, ErrNotFound
}

// FindByCWD returns the workspace registered for the given cwd, if any.
func (s *Store) FindByCWD(cwd string) (Workspace, error) {
	id := IDFor(cwd)
	return s.Get(id)
}

// Resolve looks up a workspace by exact ID, short ID prefix, or exact name.
// Returns ErrAmbiguous when a prefix or name matches multiple workspaces.
func (s *Store) Resolve(selector string) (Workspace, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if selector == "" {
		return Workspace{}, ErrNotFound
	}
	var prefixHits []Workspace
	var nameHits []Workspace
	for _, w := range s.workspaces {
		if w.ID == selector {
			return w, nil
		}
		if len(selector) < len(w.ID) && w.ID[:len(selector)] == selector {
			prefixHits = append(prefixHits, w)
		}
		if w.Name == selector {
			nameHits = append(nameHits, w)
		}
	}
	switch {
	case len(prefixHits)+len(nameHits) == 0:
		return Workspace{}, ErrNotFound
	case len(prefixHits) == 1 && len(nameHits) == 0:
		return prefixHits[0], nil
	case len(nameHits) == 1 && len(prefixHits) == 0:
		return nameHits[0], nil
	case len(prefixHits) == 1 && len(nameHits) == 1 && prefixHits[0].ID == nameHits[0].ID:
		return prefixHits[0], nil
	default:
		return Workspace{}, ErrAmbiguous
	}
}

// Add inserts a new workspace. Returns ErrExists if one is already registered
// for the same cwd.
func (s *Store) Add(w Workspace) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, existing := range s.workspaces {
		if existing.ID == w.ID {
			return ErrExists
		}
	}
	if w.SessionIDs == nil {
		w.SessionIDs = []string{}
	}
	s.workspaces = append(s.workspaces, w)
	return s.save()
}

// Remove deletes the workspace with the given ID.
func (s *Store) Remove(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, w := range s.workspaces {
		if w.ID == id {
			s.workspaces = append(s.workspaces[:i], s.workspaces[i+1:]...)
			return s.save()
		}
	}
	return ErrNotFound
}

// Touch updates LastUsedAt to now for the given workspace ID.
func (s *Store) Touch(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.workspaces {
		if s.workspaces[i].ID == id {
			s.workspaces[i].LastUsedAt = time.Now().UTC()
			return s.save()
		}
	}
	return ErrNotFound
}

// EnsureForCWD returns the workspace for cwd, creating one if missing. The
// `created` return is true when a new workspace was inserted. LastUsedAt is
// refreshed in either case.
func (s *Store) EnsureForCWD(cwd string) (Workspace, bool, error) {
	abs, err := filepath.Abs(cwd)
	if err != nil {
		return Workspace{}, false, fmt.Errorf("abs %s: %w", cwd, err)
	}
	id := IDFor(abs)
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC()
	for i := range s.workspaces {
		if s.workspaces[i].ID == id {
			s.workspaces[i].LastUsedAt = now
			if err := s.save(); err != nil {
				return Workspace{}, false, err
			}
			return s.workspaces[i], false, nil
		}
	}
	w := New(abs, "")
	w.LastUsedAt = now
	s.workspaces = append(s.workspaces, w)
	if err := s.save(); err != nil {
		return Workspace{}, false, err
	}
	return w, true, nil
}
