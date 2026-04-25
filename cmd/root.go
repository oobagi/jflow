package cmd

import (
	tea "charm.land/bubbletea/v2"
	"github.com/oobagi/jflow/internal/ui"
	"github.com/spf13/cobra"
)

const Version = "0.0.1-dev"

var debugFlag bool

var rootCmd = &cobra.Command{
	Use:           "jflow",
	Short:         "TUI harness for the claude CLI",
	Long:          "jflow drives the claude CLI as a subprocess and provides a TUI for chat, tool-driven sessions, and orchestrated multi-step work.",
	Version:       Version,
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(c *cobra.Command, args []string) error {
		m := ui.NewApp(debugFlag, Version)
		p := tea.NewProgram(m)
		_, err := p.Run()
		return err
	},
}

func init() {
	rootCmd.PersistentFlags().BoolVar(&debugFlag, "debug", false, "log verbose meta entries (key events, etc.) into the session log")
	rootCmd.SetVersionTemplate("jflow {{.Version}}\n")
	rootCmd.CompletionOptions.DisableDefaultCmd = true
}

func Execute() error {
	return rootCmd.Execute()
}
