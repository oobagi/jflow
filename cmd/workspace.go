package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	"github.com/oobagi/jflow/internal/workspace"
)

var workspaceCmd = &cobra.Command{
	Use:   "workspace",
	Short: "Manage cwd-keyed workspaces",
}

var workspaceLsCmd = &cobra.Command{
	Use:   "ls",
	Short: "List workspaces",
	RunE: func(c *cobra.Command, args []string) error {
		store, err := workspace.OpenDefault()
		if err != nil {
			return err
		}
		ws := store.List()
		w := tabwriter.NewWriter(c.OutOrStdout(), 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "ID\tNAME\tCWD\tLAST USED\tSESSIONS")
		for _, x := range ws {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%d\n",
				shortID(x.ID),
				x.Name,
				homeShorten(x.CWD),
				humanAgo(x.LastUsedAt),
				len(x.SessionIDs),
			)
		}
		return w.Flush()
	},
}

var workspaceAddName string

var workspaceAddCmd = &cobra.Command{
	Use:   "add [path]",
	Short: "Register a workspace for a directory (defaults to cwd)",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(c *cobra.Command, args []string) error {
		path := ""
		if len(args) == 1 {
			path = args[0]
		}
		if path == "" {
			cwd, err := os.Getwd()
			if err != nil {
				return err
			}
			path = cwd
		}
		abs, err := filepath.Abs(path)
		if err != nil {
			return err
		}
		store, err := workspace.OpenDefault()
		if err != nil {
			return err
		}
		w := workspace.New(abs, workspaceAddName)
		if err := store.Add(w); err != nil {
			if errors.Is(err, workspace.ErrExists) {
				return fmt.Errorf("workspace already exists for %s", abs)
			}
			return err
		}
		fmt.Fprintf(c.OutOrStdout(), "added %s  %s  %s\n", shortID(w.ID), w.Name, abs)
		return nil
	},
}

var workspaceRmCmd = &cobra.Command{
	Use:   "rm <id-or-name>",
	Short: "Remove a workspace by short ID, full ID, or name",
	Args:  cobra.ExactArgs(1),
	RunE: func(c *cobra.Command, args []string) error {
		store, err := workspace.OpenDefault()
		if err != nil {
			return err
		}
		w, err := store.Resolve(args[0])
		if err != nil {
			if errors.Is(err, workspace.ErrAmbiguous) {
				return fmt.Errorf("%q matches multiple workspaces; use the full ID", args[0])
			}
			if errors.Is(err, workspace.ErrNotFound) {
				return fmt.Errorf("no workspace matching %q", args[0])
			}
			return err
		}
		if err := store.Remove(w.ID); err != nil {
			return err
		}
		fmt.Fprintf(c.OutOrStdout(), "removed %s  %s\n", shortID(w.ID), w.Name)
		return nil
	},
}

func init() {
	workspaceAddCmd.Flags().StringVar(&workspaceAddName, "name", "", "human-readable name (defaults to dir basename)")
	workspaceCmd.AddCommand(workspaceLsCmd, workspaceAddCmd, workspaceRmCmd)
	rootCmd.AddCommand(workspaceCmd)
}

// shortID returns the first 8 chars of a workspace ID for display.
func shortID(id string) string {
	if len(id) <= 8 {
		return id
	}
	return id[:8]
}

// homeShorten replaces a leading $HOME with `~`.
func homeShorten(p string) string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return p
	}
	if p == home {
		return "~"
	}
	if strings.HasPrefix(p, home+string(os.PathSeparator)) {
		return "~" + p[len(home):]
	}
	return p
}

// humanAgo renders a relative time like "3m ago" / "2d ago" / "just now".
func humanAgo(t time.Time) string {
	if t.IsZero() {
		return "-"
	}
	d := time.Since(t)
	switch {
	case d < 30*time.Second:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	}
}
