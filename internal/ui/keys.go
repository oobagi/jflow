package ui

// Keybinding labels used in the help footer / overlay.
const (
	KeySend      = "⏎"    // enter — sends the composer
	KeyNewline   = "⇧⏎"  // shift+enter (or ctrl+j) — soft newline
	KeyCompact   = "⌃K"   // sends "/compact" as next user message
	KeyInterrupt = "⌃C"   // cancels in-flight turn (or pending spawn); never quits
	KeyQuit      = "esc"  // quit when idle; interrupt when busy; cancel recall when active
	KeyHelp      = "?"    // toggles help overlay
	KeyHistory   = "↑/↓" // recalls previous/next user message (#29)
	KeyTab       = "⇥"    // swap focus between composer and the tree pane
	KeyTreeOpen  = "⏎"    // tree: toggle workspace / activate session
	KeyTreeNew   = "n"    // tree: new session in workspace
	KeyTreeAdd   = "a"    // tree: add a new workspace (path prompt)
	KeyTreeDel   = "x"    // tree: delete (y/N confirm)
)

// HelpRow is one entry in the help footer.
type HelpRow struct{ Key, Desc string }

// ChatHelp lists keybinds that work while the composer has focus — i.e. the
// keys that actually do something when the user types in the chat. Tree-only
// keys (n/a/x/⏎-on-row) are excluded; pressing them in chat just types the
// letter, so showing them here is misleading.
func ChatHelp() []HelpRow {
	return []HelpRow{
		{KeySend, "send"},
		{KeyNewline, "newline"},
		{KeyHistory, "recall"},
		{KeyInterrupt, "interrupt"},
		{KeyCompact, "compact"},
		{KeyTab, "focus tree"},
		{KeyHelp, "help"},
		{KeyQuit, "quit"},
	}
}

// TreeHelp lists keybinds that work while the tree pane has focus.
func TreeHelp() []HelpRow {
	return []HelpRow{
		{KeyTreeOpen, "open / activate"},
		{KeyTreeNew, "new session"},
		{KeyTreeAdd, "add workspace"},
		{KeyTreeDel, "delete"},
		{"↑/↓", "move"},
		{"←/→", "collapse / expand"},
		{KeyTab, "back to chat"},
		{KeyHelp, "help"},
		{KeyQuit, "quit"},
	}
}

// RightHelp lists keybinds while the right (status) pane has focus. No
// actions are wired yet — only Tab/esc move out — so the list is intentionally
// short and labels the pane as read-only.
func RightHelp() []HelpRow {
	return []HelpRow{
		{KeyTab, "next pane"},
		{"⇧⇥", "prev pane"},
		{KeyQuit, "back to chat"},
		{KeyHelp, "help"},
	}
}

// DefaultHelp is kept as a back-compat alias for the help overlay; it picks
// the appropriate set based on whether the tree currently has focus. Callers
// should pass an *App and call the variant they want directly.
func DefaultHelp() []HelpRow { return ChatHelp() }
