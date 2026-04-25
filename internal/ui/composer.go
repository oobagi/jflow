package ui

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/bubbles/v2/textarea"
)

// Composer wraps a textarea for multiline user input.
//
// Prompt is intentionally empty — the textarea would otherwise repeat the
// prompt on every wrapped/empty line. The visible "> " cue is rendered
// once by the surrounding chrome (App.View) so multi-line input doesn't
// look like multiple prompts stacked on top of each other.
type Composer struct {
	ta textarea.Model
}

func NewComposer() Composer {
	t := textarea.New()
	t.Placeholder = "Send a message..."
	t.Prompt = ""
	t.ShowLineNumbers = false
	t.CharLimit = 0
	t.SetWidth(80)
	t.SetHeight(3)
	t.Focus()
	return Composer{ta: t}
}

// SetWidth resizes the composer.
func (c *Composer) SetWidth(w int) { c.ta.SetWidth(w) }

// SetHeight resizes the composer's vertical capacity.
func (c *Composer) SetHeight(h int) { c.ta.SetHeight(h) }

// View renders the composer.
func (c Composer) View() string { return c.ta.View() }

// Value returns the current text.
func (c Composer) Value() string { return c.ta.Value() }

// Reset clears the composer.
func (c *Composer) Reset() { c.ta.Reset() }

// Update handles textarea events.
func (c *Composer) Update(msg tea.Msg) (Composer, tea.Cmd) {
	var cmd tea.Cmd
	c.ta, cmd = c.ta.Update(msg)
	return *c, cmd
}

// InsertNewline forwards a synthetic Enter to the textarea so it inserts
// a newline at the cursor (used for shift+enter / ctrl+j bindings).
func (c *Composer) InsertNewline() tea.Cmd {
	var cmd tea.Cmd
	c.ta, cmd = c.ta.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	return cmd
}

// IsEmpty returns true if the composer has no content.
func (c Composer) IsEmpty() bool {
	return strings.TrimSpace(c.ta.Value()) == ""
}
