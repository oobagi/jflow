package ui

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/bubbles/v2/textarea"
	"charm.land/lipgloss/v2"
)

// composerMaxRows caps the composer's visible viewport so it never eats
// more than ~10 visual rows of the chat column. Beyond this the textarea
// scrolls internally rather than pushing the transcript off-screen — the
// content itself isn't capped, only the viewport.
const composerMaxRows = 10

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

	// Strip the textarea's stock cursor-row highlight (it painted a solid
	// block under the focused line) and reset the placeholder so it stays
	// dim against whatever the surrounding terminal background is.
	plain := lipgloss.NewStyle()
	placeholder := lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	styles := t.Styles()
	styles.Focused.CursorLine = plain
	styles.Focused.CursorLineNumber = plain
	styles.Focused.Placeholder = placeholder
	styles.Blurred.CursorLine = plain
	styles.Blurred.CursorLineNumber = plain
	styles.Blurred.Placeholder = placeholder
	t.SetStyles(styles)

	// Auto-grow with content (counting soft wraps) up to composerMaxRows.
	// MaxContentHeight is intentionally left unset — capping it would block
	// further input once the viewport fills up. MaxHeight only caps the
	// visible viewport, so longer content scrolls internally.
	t.DynamicHeight = true
	t.MinHeight = 1
	t.MaxHeight = composerMaxRows

	t.SetWidth(80)
	t.SetHeight(1)
	t.Focus()
	return Composer{ta: t}
}

// SetWidth resizes the composer. With DynamicHeight enabled, this also
// re-runs the wrap calculation so Height() reflects the new soft-wrap.
func (c *Composer) SetWidth(w int) { c.ta.SetWidth(w) }

// SetMaxHeight caps the composer's auto-grown height (in visual rows,
// soft wraps included). App calls this on resize so that on a very short
// terminal the composer doesn't crowd out the transcript.
func (c *Composer) SetMaxHeight(h int) {
	if h < 1 {
		h = 1
	}
	c.ta.MaxHeight = h
}

// Height returns the composer's current visual row count (auto-computed
// from content + width, capped by the configured max).
func (c Composer) Height() int { return c.ta.Height() }

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
