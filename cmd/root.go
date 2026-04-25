package cmd

import (
	"os"

	tea "charm.land/bubbletea/v2"
	"github.com/spf13/cobra"

	"github.com/oobagi/jflow/internal/config"
	"github.com/oobagi/jflow/internal/session"
	"github.com/oobagi/jflow/internal/ui"
	"github.com/oobagi/jflow/internal/workspace"
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
		wsStore, wsID := bootstrapWorkspace()
		sessStore, _ := session.OpenDefault()
		prefs, _ := config.LoadDefault()
		m := ui.NewApp(debugFlag, Version, wsStore, sessStore, prefs, wsID)
		p := tea.NewProgram(m)
		_, err := p.Run()
		return err
	},
}

// bootstrapWorkspace opens the workspace store and ensures a workspace for the
// launch cwd. Failures are non-fatal — a nil store / empty ID is allowed so
// the TUI still launches; the left pane just renders empty.
func bootstrapWorkspace() (*workspace.Store, string) {
	store, err := workspace.OpenDefault()
	if err != nil {
		return nil, ""
	}
	cwd, err := os.Getwd()
	if err != nil {
		return store, ""
	}
	w, _, err := store.EnsureForCWD(cwd)
	if err != nil {
		return store, ""
	}
	return store, w.ID
}

func init() {
	rootCmd.PersistentFlags().BoolVar(&debugFlag, "debug", false, "log verbose meta entries (key events, etc.) into the session log")
	rootCmd.SetVersionTemplate("jflow {{.Version}}\n")
	rootCmd.CompletionOptions.DisableDefaultCmd = true
}

func Execute() error {
	return rootCmd.Execute()
}
