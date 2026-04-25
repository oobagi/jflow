package ui

// Keybinding strings used in the help footer.
//
// Only bindings actually wired in App.Update are listed here. v0 prototype:
// all other "documented in 04-tui-design" keys (?, n, t, w, s, /, j/k, g/G,
// ^L, ^E) require workspaces/sessions/viewport scaffolding that lands in
// later phases. When you add the binding, add it here.
const (
	KeySend      = "⏎"        // enter — sends the composer
	KeyNewline   = "⇧⏎"      // shift+enter (or ctrl+j) — soft newline
	KeyCompact   = "⌃K"       // sends "/compact" as next user message
	KeyInterrupt = "⌃X"       // SIGINT to current claude subprocess
	KeyQuit      = "esc"      // (or ⌃C with no driver running)
)

// HelpRow is one entry in the help footer.
type HelpRow struct{ Key, Desc string }

func DefaultHelp() []HelpRow {
	return []HelpRow{
		{KeySend, "send"},
		{KeyNewline, "newline"},
		{KeyInterrupt, "interrupt"},
		{KeyCompact, "compact"},
		{KeyQuit, "quit"},
	}
}
