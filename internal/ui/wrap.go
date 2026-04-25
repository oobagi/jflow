package ui

import (
	"strings"

	"charm.land/lipgloss/v2"
	rw "github.com/mattn/go-runewidth"
	"github.com/rivo/uniseg"
)

// wrapToWidth word-wraps s to width display columns. '\n' is treated as a hard
// break. Words longer than width are hard-broken at grapheme boundaries.
// Internal runs of whitespace are collapsed to single spaces (chat-text style).
func wrapToWidth(s string, width int) string {
	if width <= 0 {
		return s
	}
	var lines []string
	for _, raw := range strings.Split(s, "\n") {
		if uniseg.StringWidth(raw) <= width {
			lines = append(lines, raw)
			continue
		}
		var cur strings.Builder
		curW := 0
		for _, w := range strings.Fields(raw) {
			wW := uniseg.StringWidth(w)
			sepW := 0
			if cur.Len() > 0 {
				sepW = 1
			}
			if curW+sepW+wW <= width {
				if sepW == 1 {
					cur.WriteByte(' ')
					curW++
				}
				cur.WriteString(w)
				curW += wW
				continue
			}
			// word doesn't fit on current line.
			if cur.Len() > 0 {
				lines = append(lines, cur.String())
				cur.Reset()
				curW = 0
			}
			if wW <= width {
				cur.WriteString(w)
				curW = wW
				continue
			}
			// word longer than width — hard-break at rune boundaries.
			for _, r := range w {
				rWidth := rw.RuneWidth(r)
				if curW+rWidth > width {
					lines = append(lines, cur.String())
					cur.Reset()
					curW = 0
				}
				cur.WriteRune(r)
				curW += rWidth
			}
		}
		if cur.Len() > 0 {
			lines = append(lines, cur.String())
		} else if len(lines) == 0 || (len(lines) > 0 && lines[len(lines)-1] != "") {
			// preserve an empty-string line for blank inputs
		}
		if len(strings.Fields(raw)) == 0 {
			lines = append(lines, "")
		}
	}
	return strings.Join(lines, "\n")
}

// wrapWithPrefix wraps s to width, then prepends `prefix` to the first line
// and `cont` to each continuation line. Returns the combined string.
// prefix and cont should be the same *visible* width (so continuation lines
// align under the first line). lipgloss.Width is used for measurement so
// ANSI escape sequences inside `prefix` are correctly treated as zero-width.
func wrapWithPrefix(s, prefix, cont string, width int) string {
	prefixW := lipgloss.Width(prefix)
	body := wrapToWidth(s, width-prefixW)
	parts := strings.Split(body, "\n")
	out := make([]string, 0, len(parts))
	for i, p := range parts {
		if i == 0 {
			out = append(out, prefix+p)
		} else {
			out = append(out, cont+p)
		}
	}
	return strings.Join(out, "\n")
}

