package session

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

const fileVersion = 1

type fileShape struct {
	Version  int       `json:"version"`
	Sessions []Session `json:"sessions"`
}

// Store wraps ~/.jflow/state/sessions.json. Single mutex; atomic save via
// temp+rename. Mirrors the workspace.Store pattern.
type Store struct {
	mu       sync.Mutex
	path     string
	sessions []Session
}

var (
	ErrNotFound  = errors.New("session not found")
	ErrAmbiguous = errors.New("ambiguous session selector")
)

// DefaultPath returns ~/.jflow/state/sessions.json.
func DefaultPath() (string, error) {
	dir, err := state.StateDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "sessions.json"), nil
}

func Open(path string) (*Store, error) {
	s := &Store{path: path}
	if err := s.load(); err != nil {
		return nil, err
	}
	return s, nil
}

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
			s.sessions = nil
			return nil
		}
		return fmt.Errorf("read %s: %w", s.path, err)
	}
	if len(data) == 0 {
		s.sessions = nil
		return nil
	}
	var f fileShape
	if err := json.Unmarshal(data, &f); err != nil {
		return fmt.Errorf("unmarshal %s: %w", s.path, err)
	}
	s.sessions = f.Sessions
	return nil
}

func (s *Store) save() error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", filepath.Dir(s.path), err)
	}
	f := fileShape{Version: fileVersion, Sessions: s.sessions}
	if f.Sessions == nil {
		f.Sessions = []Session{}
	}
	data, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal sessions: %w", err)
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

// List returns a copy of every session.
func (s *Store) List() []Session {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Session, len(s.sessions))
	copy(out, s.sessions)
	return out
}

// ListByWorkspace returns the sessions belonging to wsID, in insertion order.
func (s *Store) ListByWorkspace(wsID string) []Session {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []Session
	for _, sess := range s.sessions {
		if sess.WorkspaceID == wsID {
			out = append(out, sess)
		}
	}
	return out
}

func (s *Store) Get(id string) (Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, sess := range s.sessions {
		if sess.ID == id {
			return sess, nil
		}
	}
	return Session{}, ErrNotFound
}

// Add inserts a new session.
func (s *Store) Add(sess Session) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, existing := range s.sessions {
		if existing.ID == sess.ID {
			return fmt.Errorf("session %s already exists", sess.ID)
		}
	}
	s.sessions = append(s.sessions, sess)
	return s.save()
}

// Remove deletes the session with id.
func (s *Store) Remove(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, sess := range s.sessions {
		if sess.ID == id {
			s.sessions = append(s.sessions[:i], s.sessions[i+1:]...)
			return s.save()
		}
	}
	return ErrNotFound
}

// Touch refreshes LastUsedAt for the session with id.
func (s *Store) Touch(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.sessions {
		if s.sessions[i].ID == id {
			s.sessions[i].LastUsedAt = time.Now().UTC()
			return s.save()
		}
	}
	return ErrNotFound
}

// Rename updates the human-readable name of a session.
func (s *Store) Rename(id, name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.sessions {
		if s.sessions[i].ID == id {
			s.sessions[i].Name = name
			return s.save()
		}
	}
	return ErrNotFound
}
