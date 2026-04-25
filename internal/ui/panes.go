package ui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
)

// Pane sizing — kept here so layout decisions live in one place.
const (
	leftPaneWidth  = 30
	rightPaneWidth = 32
	// minCenterWidth is the smallest acceptable chat-column width before the
	// side panes collapse and the chat takes the whole screen.
	minCenterWidth = 40
)

// paneLayout returns the column widths for a given total width. Panes sit
// flush against the chat column — the colour contrast between Pane and
// Center backgrounds is what visually separates them. When the terminal is
// too narrow, both sidebars collapse and the chat takes the whole width.
func paneLayout(total int) (left, center, right int) {
	if total < leftPaneWidth+rightPaneWidth+minCenterWidth {
		return 0, total, 0
	}
	return leftPaneWidth, total - leftPaneWidth - rightPaneWidth, rightPaneWidth
}

// renderLeftPane draws the file-browser-style sidebar — workspaces are
// folders (▸ collapsed, ▾ expanded) and sessions appear indented under
// their parent when expanded. The cursor highlights one row when the pane
// has focus; a footer hint adapts to context (delete confirm, focused, idle).
func renderLeftPane(a *App, width, height int) string {
	if width <= 0 || height <= 0 {
		return ""
	}
	content := renderLeftBody(a, width, height)
	return renderTitledBox(a.theme, flameTitle("jflow"), width, height, content, a.focus == focusLeft)
}

// renderLeftBody builds the body of the left pane: the flattened tree, then
// a blank, then a single-line hint footer pinned to the bottom.
func renderLeftBody(a *App, width, height int) string {
	innerW := width - 6
	if innerW < 6 {
		innerW = 6
	}
	innerH := height - 4
	if innerH < 4 {
		innerH = 4
	}

	// The sidebar shows only the tree itself plus an optional delete-confirm
	// banner — the global hint bar at the bottom of the chat column carries
	// per-focus keybind hints, so the pane stays clean.
	body := renderTreeRows(a, innerW, innerH)
	if a.treeConfirmDel {
		body += "\n\n" + a.theme.StatusBad.Render("delete? y/N")
	}
	return body
}

// renderTreeRows formats the flattened tree (workspaces + their expanded
// sessions) as up to maxRows visible lines. Scrolling keeps the cursor in
// view: when treeCursor would fall off the bottom the window slides down.
func renderTreeRows(a *App, innerW, maxRows int) string {
	rows := a.tree()
	if len(rows) == 0 {
		hint := a.theme.Dim.Render("(no workspaces)")
		if a.focus == focusLeft {
			hint += "\n" + a.theme.Dim.Render("a to add")
		}
		return hint
	}
	start := 0
	if a.focus == focusLeft && a.treeCursor >= maxRows {
		start = a.treeCursor - maxRows + 1
	}
	end := start + maxRows
	if end > len(rows) {
		end = len(rows)
	}
	var out []string
	for i := start; i < end; i++ {
		r := rows[i]
		// Visual breathing room between groups: blank line after the action
		// row when general sessions follow, and again before the first
		// workspace row. Doesn't consume cursor positions — it's purely
		// presentational and only emitted when both sides are present.
		if i > start {
			prev := rows[i-1]
			if prev.kind == "action" && r.kind != "ws" {
				out = append(out, "")
			}
			if (prev.kind == "general" || prev.kind == "action") && r.kind == "ws" {
				out = append(out, "")
			}
		}
		line := renderTreeRow(a, r, innerW)
		if a.focus == focusLeft && i == a.treeCursor {
			pad := innerW - lipgloss.Width(line)
			if pad > 0 {
				line += strings.Repeat(" ", pad)
			}
			line = lipgloss.NewStyle().Reverse(true).Render(line)
		}
		out = append(out, line)
	}
	return strings.Join(out, "\n")
}

// renderTreeRow renders one tree row into innerW visual columns.
//   - "action":  "+ new session" — accent-colored, pinned at the top
//   - "general": session at the top level (default-dir chat), no indent
//   - "ws":      workspace folder with ▸/▾ caret and session-count tail
//   - "sess":    session indented inside an expanded workspace
func renderTreeRow(a *App, r treeRow, innerW int) string {
	switch r.kind {
	case "action":
		label := truncateWidth("+ new session", innerW)
		return a.theme.Accent.Render(label)
	case "general":
		marker := a.theme.Dim.Render("○ ")
		nameStyle := a.theme.Fg
		if r.sessionID == a.sessionUUID {
			marker = a.theme.Accent.Render("◉ ")
			nameStyle = a.theme.Fg.Bold(true)
		} else {
			nameStyle = a.theme.Dim
		}
		name := truncateWidth(r.sessName, innerW-2)
		return marker + nameStyle.Render(name)
	case "ws":
		caret := "▸ "
		if r.wsExpanded {
			caret = "▾ "
		}
		isLaunch := r.workspaceID == a.workspaceID
		nameStyle := a.theme.Fg
		caretStyle := a.theme.Dim
		if isLaunch {
			caretStyle = a.theme.Accent
			nameStyle = a.theme.Fg.Bold(true)
		}
		badge := ""
		nameMaxW := innerW - 2
		if r.sessCount > 0 && !r.wsExpanded {
			badge = " " + a.theme.Dim.Render(fmt.Sprintf("(%d)", r.sessCount))
			nameMaxW -= lipgloss.Width(badge)
		}
		if nameMaxW < 3 {
			nameMaxW = 3
		}
		name := truncateWidth(r.wsName, nameMaxW)
		return caretStyle.Render(caret) + nameStyle.Render(name) + badge
	case "sess":
		marker := a.theme.Dim.Render("○ ")
		nameStyle := a.theme.Dim
		if r.sessionID == a.sessionUUID {
			marker = a.theme.Accent.Render("◉ ")
			nameStyle = a.theme.Fg
		}
		name := truncateWidth(r.sessName, innerW-4)
		return "  " + marker + nameStyle.Render(name)
	}
	return ""
}

// truncateWidth clips s to maxW visual columns (ANSI-aware via lipgloss.Width),
// appending "…" if truncated. Distinct from tooluse.go's byte-length truncate.
func truncateWidth(s string, maxW int) string {
	if maxW <= 0 {
		return ""
	}
	if lipgloss.Width(s) <= maxW {
		return s
	}
	if maxW == 1 {
		return "…"
	}
	runes := []rune(s)
	for i := len(runes); i > 0; i-- {
		candidate := string(runes[:i]) + "…"
		if lipgloss.Width(candidate) <= maxW {
			return candidate
		}
	}
	return "…"
}

// flameTitle renders s with a red→yellow ember gradient — used for the
// jflow header so the brand mark looks like a flame.
func flameTitle(s string) string {
	palette := []string{"196", "202", "208", "214", "220"}
	runes := []rune(s)
	var sb strings.Builder
	for i, r := range runes {
		c := palette[i%len(palette)]
		sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color(c)).Bold(true).Render(string(r)))
	}
	return sb.String()
}

// renderTitledBox draws a rounded-border pane with `title` embedded in the
// top edge, e.g.  ╭─ jflow ─────╮ . The title may already carry ANSI styles
// (used by flameTitle) — only the surrounding spaces and ─ runs are styled
// here. Inside, the content is laid out with one row of vertical padding
// and two columns of horizontal padding so nested content has the same
// breathing room as the previous box style. Content shorter than the
// available rows is bottom-padded with blank lines.
func renderTitledBox(theme Theme, title string, width, height int, content string, focused bool) string {
	if width < 8 || height < 4 {
		return ""
	}
	border := theme.Dim
	if focused {
		border = theme.Accent
	}
	titleStr := " " + title + " "
	titleW := lipgloss.Width(titleStr)
	const leftDash = 1
	rightDash := width - 2 - leftDash - titleW
	if rightDash < 1 {
		rightDash = 1
	}

	top := border.Render("╭"+strings.Repeat("─", leftDash)) +
		titleStr +
		border.Render(strings.Repeat("─", rightDash)+"╮")
	bot := border.Render("╰" + strings.Repeat("─", width-2) + "╯")

	const padH, padV = 2, 1
	innerW := width - 2 - 2*padH
	if innerW < 1 {
		innerW = 1
	}
	innerH := height - 2 - 2*padV
	if innerH < 0 {
		innerH = 0
	}

	contentLines := strings.Split(content, "\n")
	if len(contentLines) > innerH {
		contentLines = contentLines[:innerH]
	}
	for len(contentLines) < innerH {
		contentLines = append(contentLines, "")
	}

	side := border.Render("│")
	padRow := side + strings.Repeat(" ", 2*padH+innerW) + side

	var body []string
	for i := 0; i < padV; i++ {
		body = append(body, padRow)
	}
	for _, line := range contentLines {
		w := lipgloss.Width(line)
		rightPad := innerW - w
		if rightPad < 0 {
			line = ""
			rightPad = innerW
		}
		body = append(body, side+strings.Repeat(" ", padH)+line+strings.Repeat(" ", rightPad)+strings.Repeat(" ", padH)+side)
	}
	for i := 0; i < padV; i++ {
		body = append(body, padRow)
	}

	return top + "\n" + strings.Join(body, "\n") + "\n" + bot
}

// renderRightPane shows the session-info column: model, mode, context %,
// cost, rate-limit chip. No title row — the values speak for themselves.
func renderRightPane(a *App, width, height int) string {
	if width <= 0 || height <= 0 {
		return ""
	}
	label := a.theme.Dim
	value := a.theme.Fg

	var lines []string
	push := func(s string) { lines = append(lines, s) }

	if a.status.StatusWord != "" {
		push(a.theme.HelpKey.Render("● ") + value.Render(a.status.StatusWord))
	} else {
		push(label.Render("idle"))
	}
	push("")

	push(label.Render("model"))
	if a.model != "" {
		push(value.Render(a.model))
	} else {
		push(label.Render("—"))
	}
	push(label.Render("mode"))
	if a.permMode != "" {
		push(value.Render(a.permMode))
	} else {
		push(label.Render("—"))
	}
	push("")

	push(label.Render("context"))
	pct := 0.0
	if a.ctxWindow > 0 {
		pct = float64(a.tokens) / float64(a.ctxWindow) * 100
	}
	push(value.Render(fmt.Sprintf("%s / %s", compactInt(a.tokens), compactInt(a.ctxWindow))))
	push(ctxPctStyle(a.theme, pct).Render(fmt.Sprintf("%.0f%%", pct)))
	push("")

	push(label.Render("cost"))
	push(value.Render(fmt.Sprintf("$%.2f", a.costUSD)))

	switch a.rateState {
	case "overage":
		push("")
		push(a.theme.StatusWarn.Render("⚠ overage"))
	case "exceeded":
		push("")
		push(a.theme.StatusBad.Render("⛔ rate limit"))
	}

	return renderTitledBox(a.theme, a.theme.Accent.Render("session"), width, height, strings.Join(lines, "\n"), a.focus == focusRight)
}

// ctxPctStyle picks a color band for the context-window percentage value.
func ctxPctStyle(theme Theme, pct float64) lipgloss.Style {
	switch {
	case pct >= 80:
		return theme.StatusBad
	case pct >= 50:
		return theme.StatusWarn
	default:
		return theme.StatusOK
	}
}

// compactInt renders an integer in human-friendly form (12300 → "12.3k").
func compactInt(n int) string {
	if n < 1000 {
		return fmt.Sprintf("%d", n)
	}
	if n < 1_000_000 {
		return fmt.Sprintf("%.1fk", float64(n)/1000)
	}
	return fmt.Sprintf("%.1fM", float64(n)/1_000_000)
}

// renderHintBar produces a single-line strip pinned to the bottom of the
// chat column. It surfaces only the keybinds relevant to the focused pane,
// truncating to width when needed so it never wraps. Adapts to the focus
// state on every render.
func renderHintBar(a *App, width int) string {
	if width < 1 {
		return ""
	}
	rows := a.helpForFocus()
	sep := a.theme.Dim.Render(" · ")
	parts := make([]string, 0, len(rows))
	for _, r := range rows {
		parts = append(parts, a.theme.HelpKey.Render(r.Key)+" "+a.theme.HelpDesc.Render(r.Desc))
	}
	line := strings.Join(parts, sep)
	for lipgloss.Width(line) > width-2 && len(parts) > 1 {
		parts = parts[:len(parts)-1]
		line = strings.Join(parts, sep)
	}
	if lipgloss.Width(line) > width-2 {
		line = truncateWidth(line, width-2)
	}
	return " " + line
}

// renderHelpSheetWith produces the keybinding cheatsheet that pops up under the
// composer when `?` is pressed. It sizes naturally to its content (no
// trailing padding, no title) so it slots in below the input without
// pushing the transcript away further than necessary.
// renderHelpSheetWith is the parameterised variant — caller picks ChatHelp /
// TreeHelp based on which pane currently has focus.
func renderHelpSheetWith(theme Theme, width int, bindings []HelpRow) string {
	if width < 1 {
		width = 1
	}

	maxEntry := 0
	for _, b := range bindings {
		if w := lipgloss.Width(b.Key) + 1 + lipgloss.Width(b.Desc); w > maxEntry {
			maxEntry = w
		}
	}
	colW := maxEntry + 3
	cols := (width - 4) / colW
	if cols < 1 {
		cols = 1
	}
	if cols > len(bindings) {
		cols = len(bindings)
	}
	rows := (len(bindings) + cols - 1) / cols

	hint := theme.Dim.Render("? or esc to close")

	var sb strings.Builder
	for r := 0; r < rows; r++ {
		for c := 0; c < cols; c++ {
			idx := r + c*rows
			if idx >= len(bindings) {
				continue
			}
			b := bindings[idx]
			entry := theme.HelpKey.Render(b.Key) + " " + theme.HelpDesc.Render(b.Desc)
			sb.WriteString(entry)
			visual := lipgloss.Width(b.Key) + 1 + lipgloss.Width(b.Desc)
			if pad := colW - visual; pad > 0 {
				sb.WriteString(strings.Repeat(" ", pad))
			}
		}
		sb.WriteString("\n")
	}
	sb.WriteString(hint)

	return sb.String()
}
