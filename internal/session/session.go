package session

import (
	"time"

	"github.com/google/uuid"
)

// Session is a single claude conversation owned by a workspace. The ID is the
// claude session UUID — passed to `claude -p --session-id` on first turn and
// `claude -p --resume` on subsequent turns so the model retains context.
type Session struct {
	ID          string    `json:"id"`
	WorkspaceID string    `json:"workspace_id"`
	Name        string    `json:"name"`
	CreatedAt   time.Time `json:"created_at"`
	LastUsedAt  time.Time `json:"last_used_at"`
}

// New constructs a Session with a fresh UUID. If name is empty, it is left
// blank — the UI fills in a default like "session 1" based on the workspace's
// existing session count.
func New(workspaceID, name string) Session {
	now := time.Now().UTC()
	return Session{
		ID:          uuid.NewString(),
		WorkspaceID: workspaceID,
		Name:        name,
		CreatedAt:   now,
		LastUsedAt:  now,
	}
}
