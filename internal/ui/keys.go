package ui

// Keybinding strings used in the help footer and overlay.
//
// Only bindings actually wired in App.Update are listed here. The MVP TUI
// is a single-pane chat (Phase 1); workspace/session keys (n, t, w, s, /,
// j/k, g/G, ^L) land with the three-pane shell in later phases.
const (
	KeySend      = "⏎"    // enter — sends the composer
	KeyNewline   = "⇧⏎"  // shift+enter (or ctrl+j) — soft newline
	KeyCompact   = "⌃K"   // sends "/compact" as next user message
	KeyInterrupt = "⌃C"   // cancels in-flight turn (or pending spawn); never quits
	KeyQuit      = "esc"  // quit when idle; interrupt when busy; cancel recall when active
	KeyHelp      = "?"    // toggles full-screen help overlay
	KeyHistory   = "↑/↓" // recalls previous/next user message (#29)
)

// HelpRow is one entry in the help footer.
type HelpRow struct{ Key, Desc string }

func DefaultHelp() []HelpRow {
	return []HelpRow{
		{KeySend, "send"},
		{KeyNewline, "newline"},
		{KeyHistory, "recall"},
		{KeyInterrupt, "interrupt"},
		{KeyCompact, "compact"},
		{KeyHelp, "help"},
		{KeyQuit, "quit"},
	}
}
