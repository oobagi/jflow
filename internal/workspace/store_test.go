package workspace

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func tempStore(t *testing.T) (*Store, string) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "workspaces.json")
	s, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	return s, path
}

func TestIDStability(t *testing.T) {
	id1 := IDFor("/a/b/c")
	id2 := IDFor("/a/b/c")
	id3 := IDFor("/a/b/d")
	if id1 != id2 {
		t.Fatalf("IDFor not stable: %s vs %s", id1, id2)
	}
	if id1 == id3 {
		t.Fatalf("IDFor collided across cwds: %s == %s", id1, id3)
	}
	if len(id1) != 16 {
		t.Fatalf("expected 16-char ID, got %d", len(id1))
	}
}

func TestEnsureForCWDIdempotent(t *testing.T) {
	s, _ := tempStore(t)
	cwd := t.TempDir()
	w1, created1, err := s.EnsureForCWD(cwd)
	if err != nil {
		t.Fatalf("EnsureForCWD #1: %v", err)
	}
	if !created1 {
		t.Fatal("first EnsureForCWD should report created=true")
	}
	w2, created2, err := s.EnsureForCWD(cwd)
	if err != nil {
		t.Fatalf("EnsureForCWD #2: %v", err)
	}
	if created2 {
		t.Fatal("second EnsureForCWD should report created=false")
	}
	if w1.ID != w2.ID {
		t.Fatalf("ID changed across calls: %s vs %s", w1.ID, w2.ID)
	}
	if !w2.LastUsedAt.After(w1.LastUsedAt) && !w2.LastUsedAt.Equal(w1.LastUsedAt) {
		t.Fatalf("LastUsedAt regressed: %v -> %v", w1.LastUsedAt, w2.LastUsedAt)
	}
	if got := len(s.List()); got != 1 {
		t.Fatalf("expected 1 workspace, got %d", got)
	}
}

func TestAtomicSave(t *testing.T) {
	s, path := tempStore(t)
	cwd := t.TempDir()
	if _, _, err := s.EnsureForCWD(cwd); err != nil {
		t.Fatalf("EnsureForCWD: %v", err)
	}
	// .tmp must not linger after a successful save.
	if _, err := os.Stat(path + ".tmp"); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("temp file left behind: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	var f fileShape
	if err := json.Unmarshal(data, &f); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if f.Version != fileVersion {
		t.Fatalf("version = %d, want %d", f.Version, fileVersion)
	}
	if len(f.Workspaces) != 1 {
		t.Fatalf("workspaces len = %d, want 1", len(f.Workspaces))
	}
}

func TestAddRemoveTouchRoundTrip(t *testing.T) {
	s, path := tempStore(t)

	wA := New("/tmp/a", "alpha")
	wB := New("/tmp/b", "beta")
	if err := s.Add(wA); err != nil {
		t.Fatalf("Add A: %v", err)
	}
	if err := s.Add(wB); err != nil {
		t.Fatalf("Add B: %v", err)
	}
	// Adding the same cwd again must fail.
	if err := s.Add(wA); !errors.Is(err, ErrExists) {
		t.Fatalf("Add duplicate: got %v, want ErrExists", err)
	}

	// Touch updates LastUsedAt.
	before, err := s.Get(wA.ID)
	if err != nil {
		t.Fatalf("Get A: %v", err)
	}
	time.Sleep(2 * time.Millisecond)
	if err := s.Touch(wA.ID); err != nil {
		t.Fatalf("Touch A: %v", err)
	}
	after, err := s.Get(wA.ID)
	if err != nil {
		t.Fatalf("Get A after touch: %v", err)
	}
	if !after.LastUsedAt.After(before.LastUsedAt) {
		t.Fatalf("Touch did not advance LastUsedAt: %v -> %v", before.LastUsedAt, after.LastUsedAt)
	}

	// Reload from disk and verify state survives.
	s2, err := Open(path)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	if got := len(s2.List()); got != 2 {
		t.Fatalf("after reopen: len = %d, want 2", got)
	}

	// Resolve by short ID prefix and by name.
	if w, err := s2.Resolve(wA.ID[:6]); err != nil || w.ID != wA.ID {
		t.Fatalf("Resolve prefix: %v / %+v", err, w)
	}
	if w, err := s2.Resolve("beta"); err != nil || w.ID != wB.ID {
		t.Fatalf("Resolve name: %v / %+v", err, w)
	}

	// Remove and verify.
	if err := s2.Remove(wA.ID); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	if _, err := s2.Get(wA.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Get after remove: got %v, want ErrNotFound", err)
	}
	if got := len(s2.List()); got != 1 {
		t.Fatalf("after remove: len = %d, want 1", got)
	}
}

func TestResolveAmbiguous(t *testing.T) {
	s, _ := tempStore(t)
	// Force two workspaces whose IDs share a long prefix by crafting them
	// after the fact. We can't easily collide sha256, but we can craft IDs
	// directly since Add doesn't validate the form.
	wA := Workspace{ID: "abcd1234aaaa0000", Name: "one", CWD: "/x", SessionIDs: []string{}}
	wB := Workspace{ID: "abcd1234bbbb0000", Name: "two", CWD: "/y", SessionIDs: []string{}}
	if err := s.Add(wA); err != nil {
		t.Fatalf("Add A: %v", err)
	}
	if err := s.Add(wB); err != nil {
		t.Fatalf("Add B: %v", err)
	}
	if _, err := s.Resolve("abcd"); !errors.Is(err, ErrAmbiguous) {
		t.Fatalf("Resolve ambiguous: got %v, want ErrAmbiguous", err)
	}
	if w, err := s.Resolve("abcd1234a"); err != nil || w.ID != wA.ID {
		t.Fatalf("Resolve unique prefix: %v / %+v", err, w)
	}
}
