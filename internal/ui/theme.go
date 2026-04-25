package ui

import "charm.land/lipgloss/v2"

// Theme holds the styles used across the TUI.
type Theme struct {
	Fg         lipgloss.Style
	Dim        lipgloss.Style
	Accent     lipgloss.Style
	UserPrefix lipgloss.Style
	AsstPrefix lipgloss.Style
	Thinking   lipgloss.Style
	ToolHeader lipgloss.Style
	ToolBody   lipgloss.Style
	Error      lipgloss.Style
	StatusBar  lipgloss.Style
	StatusOK   lipgloss.Style
	StatusWarn lipgloss.Style
	StatusBad  lipgloss.Style
	ComposerBg lipgloss.Style
	HelpKey    lipgloss.Style
	HelpDesc   lipgloss.Style

	// Layout fills.
	Pane   lipgloss.Style // side panels — slightly darker than the chat column
	Center lipgloss.Style // chat column — neutral
	Box    lipgloss.Style // rounded outline used by the side panels
}

func DefaultTheme() Theme {
	return Theme{
		Fg:         lipgloss.NewStyle(),
		Dim:        lipgloss.NewStyle().Foreground(lipgloss.Color("241")),
		Accent:     lipgloss.NewStyle().Foreground(lipgloss.Color("39")),
		UserPrefix: lipgloss.NewStyle().Foreground(lipgloss.Color("51")).Bold(true),
		AsstPrefix: lipgloss.NewStyle().Foreground(lipgloss.Color("213")).Bold(true),
		Thinking:   lipgloss.NewStyle().Foreground(lipgloss.Color("244")).Italic(true),
		ToolHeader: lipgloss.NewStyle().Foreground(lipgloss.Color("226")).Bold(true),
		ToolBody:   lipgloss.NewStyle().Foreground(lipgloss.Color("250")),
		Error:      lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true),
		StatusBar:  lipgloss.NewStyle().Background(lipgloss.Color("236")).Foreground(lipgloss.Color("252")).Padding(0, 1),
		StatusOK:   lipgloss.NewStyle().Foreground(lipgloss.Color("46")),
		StatusWarn: lipgloss.NewStyle().Foreground(lipgloss.Color("214")),
		StatusBad:  lipgloss.NewStyle().Foreground(lipgloss.Color("196")),
		ComposerBg: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("241")).
			Padding(0, 1),
		HelpKey:  lipgloss.NewStyle().Foreground(lipgloss.Color("51")),
		HelpDesc: lipgloss.NewStyle().Foreground(lipgloss.Color("241")),
		// Side panels get an explicit darker bg; the chat column is left
		// transparent (terminal default), so panes read as recessed
		// sidebars without us having to fight ANSI reset codes from the
		// viewport / textarea / lipgloss interplay.
		Pane:   lipgloss.NewStyle().Background(lipgloss.Color("234")),
		Center: lipgloss.NewStyle(),
		Box: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("241")).
			Padding(1, 2),
	}
}
