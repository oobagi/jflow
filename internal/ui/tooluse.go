package ui

import (
	"encoding/json"
	"fmt"
	"strings"
)

// translateToolCall produces a friendly one-line header for a tool_use
// block (e.g. "Reading composer.go", "Searching for \"foo\"").
//
// Returns (text, true) when the tool is recognised and the returned text
// already summarises the relevant args — the caller should suppress the
// raw args body in that case. Returns (name, false) for unknown tools so
// the caller can fall back to dumping the args JSON underneath.
//
// Mid-stream the args JSON is partial and won't unmarshal; calls in that
// window get a generic verb ("Reading", "Editing") which is replaced once
// the block seals and re-renders with the full args.
func translateToolCall(name, argsJSON string) (string, bool) {
	var args map[string]any
	if s := strings.TrimSpace(argsJSON); s != "" {
		_ = json.Unmarshal([]byte(s), &args)
	}
	str := func(k string) string {
		if v, ok := args[k].(string); ok {
			return v
		}
		return ""
	}

	switch name {
	case "Bash":
		return describeBash(str("command")), true
	case "Read":
		if p := str("file_path"); p != "" {
			return "Reading " + basename(p), true
		}
		return "Reading file", true
	case "Write":
		if p := str("file_path"); p != "" {
			return "Writing " + basename(p), true
		}
		return "Writing file", true
	case "Edit":
		if p := str("file_path"); p != "" {
			return "Editing " + basename(p), true
		}
		return "Editing file", true
	case "NotebookEdit":
		if p := str("notebook_path"); p != "" {
			return "Editing " + basename(p), true
		}
		return "Editing notebook", true
	case "Grep":
		return describeGrep(args), true
	case "Glob":
		if p := str("pattern"); p != "" {
			return "Listing files matching " + p, true
		}
		return "Listing files", true
	case "WebFetch":
		if u := str("url"); u != "" {
			return "Browsing " + domainOf(u), true
		}
		return "Fetching URL", true
	case "WebSearch":
		if q := str("query"); q != "" {
			return fmt.Sprintf("Searching the web for %q", truncate(q, 60)), true
		}
		return "Searching the web", true
	case "Agent", "Task":
		return describeAgent(args), true
	case "ToolSearch":
		if q := str("query"); q != "" {
			return fmt.Sprintf("Searching tools: %q", truncate(q, 40)), true
		}
		return "Searching tools", true
	case "Skill":
		if s := str("skill"); s != "" {
			return "Running skill: " + s, true
		}
		return "Running skill", true
	case "TodoWrite":
		return "Updating task list", true
	case "TaskCreate":
		return "Creating task", true
	case "TaskList":
		return "Listing tasks", true
	case "TaskGet":
		return "Reading task", true
	case "TaskUpdate":
		return "Updating task", true
	case "TaskStop":
		return "Stopping task", true
	case "TaskOutput":
		return "Reading task output", true
	case "ExitPlanMode":
		return "Presenting plan", true
	case "EnterPlanMode":
		return "Entering plan mode", true
	case "AskUserQuestion":
		return "Asking a question", true
	case "EnterWorktree":
		return "Entering worktree", true
	case "ExitWorktree":
		return "Exiting worktree", true
	case "ScheduleWakeup":
		return "Scheduling next iteration", true
	case "PushNotification":
		return "Sending notification", true
	case "RemoteTrigger":
		return "Triggering remote hook", true
	case "Monitor":
		return "Monitoring process", true
	case "CronCreate":
		return "Scheduling cron", true
	case "CronDelete":
		return "Removing cron", true
	case "CronList":
		return "Listing crons", true
	case "KillShell":
		return "Killing shell", true
	}

	if strings.HasPrefix(name, "mcp__") {
		return describeMCP(name, args), true
	}

	return name, false
}

// describeBash renders a Bash invocation as a short verb phrase. We try
// known wrappers first (`gh`, `git`) so issue/PR/run subcommands get
// human-friendly translations like "Reading GitHub issue #42". Anything
// else falls back to a verbatim `$ cmd` for short one-liners and
// "Running <tool>" for longer commands.
func describeBash(cmd string) string {
	cmd = strings.TrimSpace(cmd)
	if cmd == "" {
		return "Running command"
	}
	parts := strings.Fields(cmd)
	if d := describeGH(parts); d != "" {
		return d
	}
	if d := describeGit(parts); d != "" {
		return d
	}
	if !strings.ContainsAny(cmd, "\n") && len(cmd) <= 60 {
		return "$ " + cmd
	}
	head := parts[0]
	switch head {
	case "go", "npm", "pnpm", "yarn", "cargo", "rtk", "docker", "kubectl":
		if len(parts) >= 2 {
			return "Running " + head + " " + parts[1]
		}
	}
	return "Running " + head
}

// describeGH translates `gh <subcommand>` invocations to human-friendly
// status lines. Returns "" when parts is not a `gh` invocation.
func describeGH(parts []string) string {
	if len(parts) < 2 || parts[0] != "gh" {
		return ""
	}
	sub := parts[1]
	third := ""
	if len(parts) >= 3 {
		third = parts[2]
	}
	switch sub {
	case "issue":
		switch third {
		case "list":
			return "Listing GitHub issues"
		case "view":
			return "Reading GitHub issue" + ghIDSuffix(parts, 3)
		case "create":
			return "Creating GitHub issue"
		case "edit":
			return "Editing GitHub issue" + ghIDSuffix(parts, 3)
		case "close":
			return "Closing GitHub issue" + ghIDSuffix(parts, 3)
		case "reopen":
			return "Reopening GitHub issue" + ghIDSuffix(parts, 3)
		case "comment":
			return "Commenting on GitHub issue" + ghIDSuffix(parts, 3)
		case "lock", "unlock", "pin", "unpin", "transfer":
			return strings.Title(third) + "ing GitHub issue" + ghIDSuffix(parts, 3)
		}
		return "Running gh issue"
	case "pr":
		switch third {
		case "list":
			return "Listing GitHub PRs"
		case "view":
			return "Reading GitHub PR" + ghIDSuffix(parts, 3)
		case "create":
			return "Opening GitHub PR"
		case "edit":
			return "Editing GitHub PR" + ghIDSuffix(parts, 3)
		case "merge":
			return "Merging GitHub PR" + ghIDSuffix(parts, 3)
		case "close":
			return "Closing GitHub PR" + ghIDSuffix(parts, 3)
		case "reopen":
			return "Reopening GitHub PR" + ghIDSuffix(parts, 3)
		case "checkout":
			return "Checking out GitHub PR" + ghIDSuffix(parts, 3)
		case "checks":
			return "Reading PR checks" + ghIDSuffix(parts, 3)
		case "comment":
			return "Commenting on GitHub PR" + ghIDSuffix(parts, 3)
		case "diff":
			return "Diffing GitHub PR" + ghIDSuffix(parts, 3)
		case "review":
			return "Reviewing GitHub PR" + ghIDSuffix(parts, 3)
		case "ready":
			return "Marking PR ready" + ghIDSuffix(parts, 3)
		}
		return "Running gh pr"
	case "run":
		switch third {
		case "list":
			return "Listing workflow runs"
		case "view":
			return "Reading workflow run" + ghIDSuffix(parts, 3)
		case "watch":
			return "Watching workflow run" + ghIDSuffix(parts, 3)
		case "rerun":
			return "Re-running workflow"
		case "cancel":
			return "Cancelling workflow run"
		case "delete":
			return "Deleting workflow run"
		}
		return "Running gh run"
	case "workflow":
		switch third {
		case "list":
			return "Listing workflows"
		case "view":
			return "Reading workflow"
		case "run":
			return "Triggering workflow"
		case "enable":
			return "Enabling workflow"
		case "disable":
			return "Disabling workflow"
		}
		return "Running gh workflow"
	case "release":
		switch third {
		case "list":
			return "Listing releases"
		case "view":
			return "Reading release"
		case "create":
			return "Cutting release"
		case "edit":
			return "Editing release"
		case "delete":
			return "Deleting release"
		case "upload":
			return "Uploading release asset"
		case "download":
			return "Downloading release asset"
		}
		return "Running gh release"
	case "repo":
		switch third {
		case "view":
			return "Reading repo info"
		case "list":
			return "Listing repos"
		case "clone":
			return "Cloning repo"
		case "create":
			return "Creating repo"
		case "fork":
			return "Forking repo"
		case "delete":
			return "Deleting repo"
		case "edit":
			return "Editing repo settings"
		}
		return "Running gh repo"
	case "api":
		if len(parts) >= 3 {
			return "Calling GitHub API " + truncate(parts[2], 40)
		}
		return "Calling GitHub API"
	case "auth":
		return "Running gh auth"
	case "browse":
		return "Opening repo in browser"
	case "search":
		if third != "" {
			return "Searching GitHub " + third
		}
		return "Searching GitHub"
	case "label":
		return "Managing GitHub labels"
	case "gist":
		return "Running gh gist"
	}
	return "Running gh " + sub
}

// describeGit translates `git <subcommand>` invocations. Returns "" when
// parts is not a `git` invocation.
func describeGit(parts []string) string {
	if len(parts) < 2 || parts[0] != "git" {
		return ""
	}
	sub := parts[1]
	switch sub {
	case "status":
		return "Checking git status"
	case "diff":
		return "Diffing"
	case "log":
		return "Reading git log"
	case "show":
		return "Reading git commit"
	case "commit":
		return "Committing"
	case "push":
		return "Pushing to remote"
	case "pull":
		return "Pulling from remote"
	case "fetch":
		return "Fetching from remote"
	case "checkout":
		return "Switching git ref"
	case "switch":
		return "Switching branch"
	case "branch":
		return "Managing branches"
	case "merge":
		return "Merging branch"
	case "rebase":
		return "Rebasing"
	case "stash":
		return "Stashing changes"
	case "add":
		return "Staging changes"
	case "rm":
		return "Removing files"
	case "mv":
		return "Renaming files"
	case "reset":
		return "Resetting"
	case "restore":
		return "Restoring files"
	case "tag":
		return "Managing tags"
	case "remote":
		return "Managing remotes"
	case "clone":
		return "Cloning repo"
	case "init":
		return "Initialising repo"
	case "blame":
		return "Reading git blame"
	case "worktree":
		return "Managing worktrees"
	case "rev-parse":
		return "Reading git ref"
	case "config":
		return "Reading git config"
	case "cherry-pick":
		return "Cherry-picking"
	case "revert":
		return "Reverting commit"
	}
	return "Running git " + sub
}

// ghIDSuffix returns " #N" when parts[i] looks like a numeric issue/PR id.
func ghIDSuffix(parts []string, i int) string {
	if i >= len(parts) {
		return ""
	}
	n := parts[i]
	if n == "" {
		return ""
	}
	for _, r := range n {
		if r < '0' || r > '9' {
			return ""
		}
	}
	return " #" + n
}

func describeGrep(args map[string]any) string {
	pattern, _ := args["pattern"].(string)
	path, _ := args["path"].(string)
	if pattern == "" {
		return "Searching"
	}
	pat := truncate(pattern, 50)
	if path != "" {
		return fmt.Sprintf("Searching %s for %q", basename(path), pat)
	}
	return fmt.Sprintf("Searching for %q", pat)
}

func describeAgent(args map[string]any) string {
	t, _ := args["subagent_type"].(string)
	d, _ := args["description"].(string)
	switch {
	case t != "" && d != "":
		return fmt.Sprintf("Spawning %s agent — %s", t, truncate(d, 50))
	case t != "":
		return "Spawning " + t + " agent"
	case d != "":
		return "Spawning agent — " + truncate(d, 50)
	}
	return "Spawning agent"
}

// describeMCP turns an `mcp__<server>__<method>` name into a friendly
// phrase by inferring an action verb from the method prefix and using
// the rest as a noun. Server names get vendor-prefix stripping
// ("claude_ai_Gmail" → "Gmail") and underscores → spaces.
func describeMCP(name string, args map[string]any) string {
	rest := strings.TrimPrefix(name, "mcp__")
	parts := strings.SplitN(rest, "__", 2)
	server := mcpServerLabel(parts[0])
	if len(parts) < 2 {
		return "Calling " + server
	}
	verb, obj := mcpVerbObject(parts[1])
	if obj == "" {
		return verb + " " + server
	}
	return verb + " " + server + " " + obj
}

func mcpServerLabel(s string) string {
	s = strings.TrimPrefix(s, "claude_ai_")
	s = strings.TrimPrefix(s, "plugin_")
	return strings.ReplaceAll(s, "_", " ")
}

// mcpVerbObject splits an MCP method like "get_issue" into ("Reading",
// "issue"). Falls back to ("Calling", method) when no prefix matches.
func mcpVerbObject(method string) (string, string) {
	m := strings.ToLower(method)
	prefixes := []struct{ p, verb string }{
		{"get_", "Reading"},
		{"read_", "Reading"},
		{"fetch_", "Reading"},
		{"query-docs", "Querying docs on"},
		{"query_docs", "Querying docs on"},
		{"query", "Querying"},
		{"search_docs", "Searching docs on"},
		{"search_", "Searching"},
		{"search-", "Searching"},
		{"list_", "Listing"},
		{"create_", "Creating"},
		{"add_", "Adding to"},
		{"update_", "Updating"},
		{"set_", "Updating"},
		{"delete_", "Deleting"},
		{"remove_", "Deleting from"},
		{"run_", "Running"},
		{"execute_", "Running"},
		{"deploy_", "Deploying"},
		{"apply_", "Applying"},
		{"resolve-", "Resolving"},
		{"resolve_", "Resolving"},
		{"navigate", "Navigating"},
		{"browser_navigate", "Navigating"},
		{"browser_click", "Clicking in"},
		{"browser_take_screenshot", "Screenshotting"},
		{"browser_snapshot", "Snapshotting"},
		{"browser_type", "Typing in"},
		{"browser_press_key", "Pressing key in"},
		{"browser_", "Browsing in"},
		{"take_screenshot", "Screenshotting"},
		{"snapshot", "Snapshotting"},
		{"open_", "Opening"},
		{"close_", "Closing"},
		{"start_", "Starting"},
		{"stop_", "Stopping"},
		{"launch_", "Launching"},
		{"input_text", "Typing in"},
		{"tap_on", "Tapping in"},
		{"check_flow_syntax", "Checking flow syntax in"},
		{"complete_authentication", "Completing auth on"},
		{"authenticate", "Authenticating with"},
		{"cheat_sheet", "Reading cheat sheet for"},
	}
	for _, p := range prefixes {
		if strings.HasPrefix(m, p.p) {
			rest := strings.TrimPrefix(method, method[:len(p.p)])
			return p.verb, strings.ReplaceAll(rest, "_", " ")
		}
	}
	return "Calling", strings.ReplaceAll(method, "_", " ")
}

func basename(p string) string {
	p = strings.TrimSpace(p)
	if i := strings.LastIndex(p, "/"); i >= 0 {
		return p[i+1:]
	}
	return p
}

func domainOf(u string) string {
	s := u
	if i := strings.Index(s, "://"); i >= 0 {
		s = s[i+3:]
	}
	if i := strings.Index(s, "/"); i >= 0 {
		s = s[:i]
	}
	return s
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	if n <= 1 {
		return "…"
	}
	return s[:n-1] + "…"
}
