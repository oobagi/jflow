package ui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
)

// Pane sizing — kept here so layout decisions live in one place.
const (
	leftPaneWidth  = 18
	rightPaneWidth = 26
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

// renderLeftPane is a stubbed workspaces sidebar — phase 2 will populate it.
// For now it just renders an empty pane with the flame-coloured "jflow"
// header inlined into the top border.
func renderLeftPane(theme Theme, width, height int) string {
	if width <= 0 || height <= 0 {
		return ""
	}
	return renderTitledBox(theme, flameTitle("jflow"), width, height, "")
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
func renderTitledBox(theme Theme, title string, width, height int, content string) string {
	if width < 8 || height < 4 {
		return ""
	}
	border := theme.Dim
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

	return renderTitledBox(a.theme, a.theme.Accent.Render("session"), width, height, strings.Join(lines, "\n"))
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

// renderHelpSheet produces the keybinding cheatsheet that pops up under the
// composer when `?` is pressed. It sizes naturally to its content (no
// trailing padding, no title) so it slots in below the input without
// pushing the transcript away further than necessary.
func renderHelpSheet(theme Theme, width int) string {
	if width < 1 {
		width = 1
	}
	bindings := DefaultHelp()

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
