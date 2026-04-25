package workspace

import (
	"crypto/sha256"
	"encoding/hex"
	"path/filepath"
	"time"
)

// Workspace is a cwd-keyed grouping of sessions. The ID is derived from the
// canonicalized cwd so it survives renames of jflow's own state files.
type Workspace struct {
	ID         string    `json:"id"`
	Name       string    `json:"name"`
	CWD        string    `json:"cwd"`
	CreatedAt  time.Time `json:"created_at"`
	LastUsedAt time.Time `json:"last_used_at"`
	SessionIDs []string  `json:"session_ids"`
}

// IDFor returns the stable workspace ID for a cwd: sha256(cwd)[:16].
func IDFor(cwd string) string {
	sum := sha256.Sum256([]byte(cwd))
	return hex.EncodeToString(sum[:])[:16]
}

// New constructs a Workspace for the given cwd. If name is empty, the basename
// of the cwd is used. Caller is responsible for canonicalizing the cwd before
// calling (Store.EnsureForCWD does this).
func New(cwd, name string) Workspace {
	if name == "" {
		name = filepath.Base(cwd)
	}
	now := time.Now().UTC()
	return Workspace{
		ID:         IDFor(cwd),
		Name:       name,
		CWD:        cwd,
		CreatedAt:  now,
		LastUsedAt: now,
		SessionIDs: []string{},
	}
}
