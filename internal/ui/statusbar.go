package ui

// StatusBar holds the running session-info state. The bottom bar that used
// to render this data is gone — values now drive the right-hand session
// panel (see panes.go renderRightPane). The struct stays as the carrier for
// the streaming status word so applyEvent can stash it in one place.
type StatusBar struct {
	Tool           string
	Workspace      string
	Model          string
	PermissionMode string
	Tokens         int
	ContextWindow  int
	CostUSD        float64
	RateStatus     string // "" | "ok" | "overage" | "exceeded"
	StatusWord     string // streaming spinner word
}
