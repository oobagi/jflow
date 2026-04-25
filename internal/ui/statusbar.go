package ui

import (
	"fmt"
	"strings"
)

// StatusBar holds the data the status bar needs to render.
type StatusBar struct {
	Tool          string
	Workspace     string
	Model         string
	PermissionMode string
	Tokens        int
	ContextWindow int
	CostUSD       float64
	RateStatus    string // "" | "ok" | "overage" | "exceeded"
	StatusWord    string // streaming spinner word
}

// View renders the status bar to a single styled string of the given width.
func (s StatusBar) View(theme Theme, width int) string {
	left := s.Tool
	if left == "" {
		left = "manual"
	}
	if s.Workspace != "" {
		left += " · " + s.Workspace
	}
	if s.Model != "" {
		left += " · " + s.Model
	}
	if s.PermissionMode != "" {
		left += " · " + s.PermissionMode
	}
	if s.StatusWord != "" {
		left += " · " + s.StatusWord
	}

	pct := 0.0
	if s.ContextWindow > 0 {
		pct = float64(s.Tokens) / float64(s.ContextWindow) * 100
	}
	right := fmt.Sprintf("%d/%d (%.0f%%) · $%.4f", s.Tokens, s.ContextWindow, pct, s.CostUSD)
	if s.RateStatus == "overage" {
		right = theme.StatusWarn.Render("⚠ overage") + " · " + right
	} else if s.RateStatus == "exceeded" {
		right = theme.StatusBad.Render("⛔ rate") + " · " + right
	}

	pad := width - lenNoStyle(left) - lenNoStyle(right)
	if pad < 1 {
		pad = 1
	}
	bar := left + strings.Repeat(" ", pad) + right
	return theme.StatusBar.Width(width).Render(bar)
}

// lenNoStyle is a naive width estimator that treats the string as plain.
// Good enough for v0; replace with lipgloss/v2.Width when we add ANSI styling.
func lenNoStyle(s string) int {
	n := 0
	for range s {
		n++
	}
	return n
}
