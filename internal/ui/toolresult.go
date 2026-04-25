package ui

import (
	"fmt"
	"strings"
)

// maxToolResultPreviewLines caps how many lines of the tool result body
// we render under the footer. Anything beyond is summarised with a
// "… (N more lines)" trailer.
const maxToolResultPreviewLines = 5

// renderToolResultFooter renders the dim footer attached to a tool_use
// block once its matching tool_result arrives. Output:
//
//	→ 12 lines
//	│ first line of stdout
//	│ second line…
//	│ … (7 more lines)
//
// The leading two-space indent matches the body indent used by tool_use
// rendering above.
func renderToolResultFooter(theme Theme, body string, isErr bool, width int) []string {
	body = strings.TrimRight(body, "\n")
	if body == "" {
		return []string{theme.Dim.Render("  → (no output)")}
	}
	lines := strings.Split(body, "\n")
	header := fmt.Sprintf("  → %s", summarizeToolResult(lines, isErr))

	style := theme.Dim
	if isErr {
		style = theme.Error
	}

	out := []string{style.Render(header)}

	preview := lines
	more := 0
	if len(preview) > maxToolResultPreviewLines {
		more = len(preview) - maxToolResultPreviewLines
		preview = preview[:maxToolResultPreviewLines]
	}
	wrapWidth := width - 4
	if wrapWidth < 10 {
		wrapWidth = 10
	}
	for _, p := range preview {
		wrapped := wrapToWidth(p, wrapWidth)
		for _, q := range strings.Split(wrapped, "\n") {
			out = append(out, theme.Dim.Render("  │ "+q))
		}
	}
	if more > 0 {
		out = append(out, theme.Dim.Render(fmt.Sprintf("  │ … (%d more line%s)", more, plural(more))))
	}
	return out
}

// summarizeToolResult returns the short header tail like "12 lines" or
// "error: 3 lines".
func summarizeToolResult(lines []string, isErr bool) string {
	n := len(lines)
	suffix := fmt.Sprintf("%d line%s", n, plural(n))
	if isErr {
		return "error · " + suffix
	}
	return suffix
}

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}
