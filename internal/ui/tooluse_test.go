package ui

import "testing"

func TestTranslateToolCall(t *testing.T) {
	cases := []struct {
		name string
		args string
		want string
		ok   bool
	}{
		{"Read", `{"file_path":"/Users/jaden/.jflow/internal/ui/composer.go"}`, "Reading composer.go", true},
		{"Edit", `{"file_path":"/tmp/x.go"}`, "Editing x.go", true},
		{"Write", `{"file_path":"new.txt"}`, "Writing new.txt", true},
		{"Grep", `{"pattern":"foo","path":"internal/ui"}`, `Searching ui for "foo"`, true},
		{"Grep", `{"pattern":"foo"}`, `Searching for "foo"`, true},
		{"Glob", `{"pattern":"**/*.go"}`, "Listing files matching **/*.go", true},
		{"Bash", `{"command":"git status"}`, "Checking git status", true},
		{"Bash", `{"command":"git log --pretty=format:%H -n 1000 && something else extra long"}`, "Reading git log", true},
		{"Bash", `{"command":"git diff --stat"}`, "Diffing", true},
		{"Bash", `{"command":"git push -u origin feat/x"}`, "Pushing to remote", true},
		{"Bash", `{"command":"gh issue list --state open --limit 50"}`, "Listing GitHub issues", true},
		{"Bash", `{"command":"gh issue view 39"}`, "Reading GitHub issue #39", true},
		{"Bash", `{"command":"gh pr view 123"}`, "Reading GitHub PR #123", true},
		{"Bash", `{"command":"gh pr create --title foo --body bar"}`, "Opening GitHub PR", true},
		{"Bash", `{"command":"gh pr merge 7 --merge"}`, "Merging GitHub PR #7", true},
		{"Bash", `{"command":"gh run watch 123"}`, "Watching workflow run #123", true},
		{"Bash", `{"command":"gh api repos/foo/bar/pulls"}`, "Calling GitHub API repos/foo/bar/pulls", true},
		{"Bash", `{"command":"go build ./..."}`, "$ go build ./...", true},
		{"WebFetch", `{"url":"https://github.com/foo/bar/issues/1"}`, "Browsing github.com", true},
		{"WebSearch", `{"query":"go textarea"}`, `Searching the web for "go textarea"`, true},
		{"TodoWrite", `{}`, "Updating task list", true},
		{"Agent", `{"subagent_type":"Explore","description":"find foo"}`, "Spawning Explore agent — find foo", true},
		{"ToolSearch", `{"query":"select:Read"}`, `Searching tools: "select:Read"`, true},

		// MCP — generic verb extraction
		{"mcp__github__get_issue", `{}`, "Reading github issue", true},
		{"mcp__playwright__browser_navigate", `{}`, "Navigating playwright", true},
		{"mcp__claude_ai_Gmail__authenticate", `{}`, "Authenticating with Gmail", true},
		{"mcp__claude_ai_Supabase__list_projects", `{}`, "Listing Supabase projects", true},

		// Unknown tool → falls through with ok=false
		{"NotARealTool", `{"x":"y"}`, "NotARealTool", false},

		// Mid-stream partial JSON → falls back to generic verb
		{"Read", `{"file_path":"/Users`, "Reading file", true},
	}

	for _, c := range cases {
		got, ok := translateToolCall(c.name, c.args)
		if got != c.want || ok != c.ok {
			t.Errorf("translateToolCall(%q, %q) = (%q, %v); want (%q, %v)", c.name, c.args, got, ok, c.want, c.ok)
		}
	}
}
