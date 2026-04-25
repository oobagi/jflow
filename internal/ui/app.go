package ui

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/bubbles/v2/textarea"
	"charm.land/bubbles/v2/viewport"
	"charm.land/lipgloss/v2"
	"github.com/google/uuid"

	"github.com/oobagi/jflow/internal/claude"
	"github.com/oobagi/jflow/internal/state"
)

// App is the root Bubble Tea model for the v0 single-pane chat prototype.
// It owns one ephemeral session, spawning a fresh `claude -p --resume <uuid>`
// subprocess per user turn. Streaming events update the transcript live.
type App struct {
	theme       Theme
	width       int
	height      int
	transcript  Transcript
	viewport    viewport.Model
	composer    Composer
	status      StatusBar
	sessionUUID string
	firstTurn   bool

	// One driver lives per turn.
	driver *claude.Driver
	cancel context.CancelFunc

	// spawning is true between the moment the user hits enter and the moment
	// claude.Spawn returns (or the spawn is cancelled). During this window
	// the UI shows "starting…" and ⌃C/esc cancel the pending spawn instead
	// of being lost to a synchronous Spawn call.
	spawning bool

	// Aggregated session usage (running totals).
	tokens    int
	ctxWindow int
	costUSD   float64
	model     string
	permMode  string
	rateState string

	// Banner text for the session (after first system/init).
	banner string

	// Worktree info shown inline with the composer rule. Detected once on
	// startup; phase 2 will refresh per-workspace.
	worktree string
	branch   string

	// showHelp toggles the full-screen help overlay (#26).
	showHelp bool

	quitting bool

	// Debug logging: always-on raw JSONL log + jflow meta entries.
	// `--debug` adds extra meta entries (key presses, etc.).
	debug      bool
	logFile    *os.File
	logPath    string
	jflowVer   string
}

// NewApp constructs the App with a fresh session uuid.
//
// `debug` enables verbose meta entries in the session log (key events, etc.).
// `version` is recorded in the session-start meta entry.
//
// Session logs are always written to ~/.jflow/state/logs/<ts>-<sid8>.jsonl
// regardless of debug mode, and a `last.jsonl` symlink is updated on session
// start so future runs (or claude itself) can find the most recent session.
func NewApp(debug bool, version string) *App {
	a := &App{
		theme:       DefaultTheme(),
		viewport:    viewport.New(),
		composer:    NewComposer(),
		sessionUUID: uuid.NewString(),
		firstTurn:   true,
		ctxWindow:   200000, // sane default until the first `result` updates it
		debug:       debug,
		jflowVer:    version,
	}
	a.viewport.SoftWrap = true
	a.worktree, a.branch = detectWorktreeBranch()
	a.openLog()
	return a
}

// detectWorktreeBranch returns the home-shortened cwd and the current git
// branch (or empty if not a git repo / detached HEAD that returns "HEAD").
func detectWorktreeBranch() (string, string) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", ""
	}
	if home, err := os.UserHomeDir(); err == nil {
		if cwd == home {
			cwd = "~"
		} else if strings.HasPrefix(cwd, home+string(os.PathSeparator)) {
			cwd = "~" + cwd[len(home):]
		}
	}
	branch := ""
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	if out, err := cmd.Output(); err == nil {
		b := strings.TrimSpace(string(out))
		if b != "" && b != "HEAD" {
			branch = b
		}
	}
	return cwd, branch
}

// openLog creates the session log file and updates the `last.jsonl` symlink.
// Failures are logged into the transcript banner but do not abort startup.
func (a *App) openLog() {
	dir, err := state.LogsDir()
	if err != nil {
		a.transcript.AddSystemNote("log dir unavailable: " + err.Error())
		return
	}
	name := time.Now().UTC().Format("20060102T150405Z") + "-" + a.sessionUUID[:8] + ".jsonl"
	path := filepath.Join(dir, name)
	f, err := os.Create(path)
	if err != nil {
		a.transcript.AddSystemNote("log open failed: " + err.Error())
		return
	}
	a.logFile = f
	a.logPath = path

	last := filepath.Join(dir, "last.jsonl")
	_ = os.Remove(last)
	_ = os.Symlink(path, last)

	a.writeMeta("session_start", map[string]any{
		"session_uuid":  a.sessionUUID,
		"jflow_version": a.jflowVer,
		"debug":         a.debug,
	})
}

// writeMeta appends a `_jflow` meta entry to the session log. Each entry has
// `_jflow` (kind), `ts` (RFC3339Nano UTC), plus the supplied fields.
func (a *App) writeMeta(kind string, fields map[string]any) {
	if a.logFile == nil {
		return
	}
	if fields == nil {
		fields = map[string]any{}
	}
	fields["_jflow"] = kind
	fields["ts"] = time.Now().UTC().Format(time.RFC3339Nano)
	b, err := json.Marshal(fields)
	if err != nil {
		return
	}
	_, _ = a.logFile.Write(b)
	_, _ = a.logFile.Write([]byte{'\n'})
}

// LogPath returns the absolute path of the session log file (empty if logging failed).
func (a *App) LogPath() string { return a.logPath }

// Init satisfies tea.Model. Returns textarea.Blink so the composer cursor
// is properly managed (no double-cursor artifacts).
func (a *App) Init() tea.Cmd {
	a.transcript.AddSystemNote("new session " + a.sessionUUID[:8] + " — type a message and press enter")
	return textarea.Blink
}

// driverEventMsg wraps a claude event so it can be delivered as a tea.Msg.
type driverEventMsg struct{ ev claude.Event }

// driverDoneMsg is delivered when the event channel closes (after DriverExit).
type driverDoneMsg struct{}

// spawnedMsg is delivered when the off-loop claude.Spawn call returns.
// One of driver / err will be set.
type spawnedMsg struct {
	driver *claude.Driver
	err    error
}

// readDriver returns a tea.Cmd that pulls one event off the driver and emits it.
// When the channel closes, it emits driverDoneMsg.
func readDriver(d *claude.Driver) tea.Cmd {
	return func() tea.Msg {
		ev, ok := <-d.Events()
		if !ok {
			return driverDoneMsg{}
		}
		return driverEventMsg{ev: ev}
	}
}

// Update handles all events.
func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch m := msg.(type) {

	case tea.WindowSizeMsg:
		a.width = m.Width
		a.height = m.Height
		// Pane sizing is recomputed on every render in View(); nothing to
		// do here beyond stashing the dimensions.
		return a, nil

	case tea.MouseWheelMsg:
		switch m.Button {
		case tea.MouseWheelUp:
			a.viewport.ScrollUp(3)
		case tea.MouseWheelDown:
			a.viewport.ScrollDown(3)
		}
		return a, nil

	case tea.KeyPressMsg:
		key := m.String()
		// Help overlay swallows all keys except the toggles that close it.
		if a.showHelp {
			switch key {
			case "?", "esc", "ctrl+c":
				a.showHelp = false
				if a.debug {
					a.writeMeta("key", map[string]any{"key": key, "context": "help_close"})
				}
			}
			return a, nil
		}
		// Open help only when the composer is empty so the user can still
		// type "?" in a message.
		if key == "?" && a.composer.IsEmpty() {
			a.showHelp = true
			if a.debug {
				a.writeMeta("key", map[string]any{"key": key, "context": "help_open"})
			}
			return a, nil
		}
		switch key {
		case "ctrl+c":
			// Interrupt-only: never quits. Cancels pending spawn or running
			// turn; no-op when idle so mashed ⌃C can't kill the CLI.
			if a.cancelTurn(key) {
				return a, nil
			}
			return a, nil
		case "esc":
			// esc cancels a pending spawn / running turn; if idle, quits.
			if a.cancelTurn(key) {
				return a, nil
			}
			a.writeMeta("session_end", map[string]any{
				"session_uuid": a.sessionUUID,
				"reason":       "user_quit",
			})
			if a.logFile != nil {
				_ = a.logFile.Close()
			}
			a.quitting = true
			return a, tea.Quit
		case "ctrl+k":
			// compact: send "/compact" as the next user message
			return a, a.send("/compact")
		case "shift+enter", "ctrl+j":
			// soft newline — forward to the textarea as a synthetic Enter
			cmd := a.composer.InsertNewline()
			if a.debug {
				a.writeMeta("key", map[string]any{"key": key, "context": "newline"})
			}
			return a, cmd
		case "enter":
			if a.driver != nil || a.spawning {
				if a.debug {
					a.writeMeta("key", map[string]any{"key": key, "context": "ignored_busy"})
				}
				return a, nil
			}
			text := strings.TrimSpace(a.composer.Value())
			if text == "" {
				return a, nil
			}
			a.composer.Reset()
			if a.debug {
				a.writeMeta("key", map[string]any{"key": key, "context": "send"})
			}
			return a, a.send(text)
		}
		if a.debug {
			a.writeMeta("key", map[string]any{"key": key})
		}
		// Otherwise pass through to composer.
		var cmd tea.Cmd
		a.composer, cmd = a.composer.Update(msg)
		return a, cmd

	case driverEventMsg:
		a.applyEvent(m.ev)
		if a.driver != nil {
			return a, readDriver(a.driver)
		}
		return a, nil

	case driverDoneMsg:
		a.driver = nil
		a.cancel = nil
		a.status.StatusWord = ""
		return a, nil

	case spawnedMsg:
		a.spawning = false
		if m.err != nil {
			a.cancel = nil
			a.status.StatusWord = ""
			a.transcript.AddSystemNote("spawn error: " + m.err.Error())
			a.writeMeta("spawn_error", map[string]any{"err": m.err.Error()})
			return a, nil
		}
		a.driver = m.driver
		a.writeMeta("spawn", map[string]any{
			"session_uuid": a.sessionUUID,
			"argv":         m.driver.Argv(),
		})
		a.status.StatusWord = "thinking…"
		return a, readDriver(m.driver)
	}

	// Pass remaining messages to the composer.
	var cmd tea.Cmd
	a.composer, cmd = a.composer.Update(msg)
	return a, cmd
}

// send queues a new claude turn for the given user message. The actual
// claude.Spawn happens off the Bubble Tea event loop so the UI stays
// responsive (and ⌃C/esc can cancel the pending spawn).
func (a *App) send(text string) tea.Cmd {
	a.transcript.AddUserMessage(text)
	a.writeMeta("user_send", map[string]any{
		"session_uuid": a.sessionUUID,
		"text":         text,
	})
	ctx, cancel := context.WithCancel(context.Background())
	a.cancel = cancel
	opts := claude.SpawnOpts{
		SessionID: a.sessionUUID,
		Resume:    !a.firstTurn,
		Prompt:    text,
		LogWriter: a.logFile,
	}
	a.firstTurn = false
	a.spawning = true
	a.status.StatusWord = "starting…"
	return func() tea.Msg {
		d, err := claude.Spawn(ctx, opts)
		return spawnedMsg{driver: d, err: err}
	}
}

// composerRule builds the dim `─` rule above the composer with the current
// worktree path and branch embedded near the left edge — like:
//   ── ~/.jflow · main ──────────────────
func (a *App) composerRule(width int) string {
	if width < 1 {
		width = 1
	}
	const leftDash = 2
	var label string
	if a.worktree != "" {
		label = " " + a.theme.Dim.Render(a.worktree)
	}
	if a.branch != "" {
		if label != "" {
			label += a.theme.Dim.Render(" · ")
		} else {
			label = " "
		}
		label += a.theme.Accent.Render(a.branch)
	}
	if label != "" {
		label += " "
	}
	labelW := lipgloss.Width(label)
	rightDash := width - leftDash - labelW
	if rightDash < 1 {
		// Not enough room — fall back to a plain rule.
		return a.theme.Dim.Render(strings.Repeat("─", width))
	}
	return a.theme.Dim.Render(strings.Repeat("─", leftDash)) +
		label +
		a.theme.Dim.Render(strings.Repeat("─", rightDash))
}

// indentLines prefixes every line of s with `cols` spaces — used to inset
// the chat column away from the side panels' borders.
func indentLines(s string, cols int) string {
	if cols <= 0 {
		return s
	}
	prefix := strings.Repeat(" ", cols)
	parts := strings.Split(s, "\n")
	for i, p := range parts {
		parts[i] = prefix + p
	}
	return strings.Join(parts, "\n")
}

// formatDuration renders a millisecond duration as a compact human string
// suitable for the transcript trailer: "230ms", "2.3s", "1m 12s".
func formatDuration(ms int64) string {
	if ms < 1000 {
		return fmt.Sprintf("%dms", ms)
	}
	secs := float64(ms) / 1000
	if secs < 60 {
		return fmt.Sprintf("%.1fs", secs)
	}
	m := int(secs) / 60
	s := int(secs) % 60
	return fmt.Sprintf("%dm %ds", m, s)
}

// cancelTurn aborts whatever turn-related work is currently running.
// Returns true if something was actively cancelled (so the caller knows
// the keypress was consumed); false when the harness is idle.
func (a *App) cancelTurn(key string) bool {
	if a.driver != nil {
		_ = a.driver.Interrupt()
		if a.debug {
			a.writeMeta("key", map[string]any{"key": key, "context": "interrupt"})
		}
		return true
	}
	if a.spawning {
		if a.cancel != nil {
			a.cancel()
		}
		if a.debug {
			a.writeMeta("key", map[string]any{"key": key, "context": "cancel_spawn"})
		}
		return true
	}
	return false
}

// applyEvent updates state from one claude event.
func (a *App) applyEvent(ev claude.Event) {
	switch e := ev.(type) {
	case claude.SystemInit:
		a.model = e.Model
		a.permMode = e.PermissionMode
		a.status.Model = e.Model
		a.status.PermissionMode = e.PermissionMode
		if a.banner == "" && e.ClaudeCodeVersion != "" {
			a.banner = fmt.Sprintf("claude %s · %s · %s", e.ClaudeCodeVersion, e.Model, e.PermissionMode)
			a.transcript.AddSystemNote(a.banner)
		}

	case claude.SystemStatus:
		a.status.StatusWord = e.Status

	case claude.HookStarted, claude.HookResponse:
		// suppressed for now; --bare will eliminate these in tool sessions

	case claude.MessageStart:
		// rolling totals updated on message_delta and result; no-op here
		_ = e

	case claude.ContentBlockStart:
		a.transcript.OnContentBlockStart(e.Index, e.Block)

	case claude.ContentBlockDelta:
		a.transcript.OnContentBlockDelta(e.Index, e.Delta)

	case claude.ContentBlockStop:
		a.transcript.OnContentBlockStop(e.Index)

	case claude.MessageDelta:
		// running token count for in-flight visibility
		a.tokens = e.Usage.Total()
		a.status.Tokens = a.tokens

	case claude.MessageStop:
		// no-op; the next event is usually `assistant` snapshot then `result`

	case claude.AssistantSnapshot:
		// could reconcile streamed deltas vs canonical snapshot here; v0 trusts deltas
		_ = e

	case claude.UserEcho:
		// only fires with --replay-user-messages; v0 does not enable that flag

	case claude.RateLimit:
		if e.Info.IsUsingOverage {
			a.rateState = "overage"
		} else if e.Info.Status != "allowed" {
			a.rateState = "exceeded"
		} else {
			a.rateState = "ok"
		}
		a.status.RateStatus = a.rateState

	case claude.Result:
		if mu, ok := e.ModelUsage[a.model]; ok && mu.ContextWindow > 0 {
			a.ctxWindow = mu.ContextWindow
			a.status.ContextWindow = mu.ContextWindow
		}
		a.costUSD += e.TotalCostUSD
		a.status.CostUSD = a.costUSD
		if e.DurationMS > 0 {
			a.transcript.AddTiming(formatDuration(e.DurationMS))
		}
		if e.IsError {
			a.transcript.AddSystemNote("error: " + e.Result + " (terminal_reason=" + e.TerminalReason + ")")
		}

	case claude.ParseError:
		a.transcript.AddSystemNote("parse error: " + e.Err.Error())

	case claude.DriverExit:
		if e.Err != nil {
			note := "subprocess exited: " + e.Err.Error()
			if e.Stderr != "" {
				note += "\n" + e.Stderr
			}
			a.transcript.AddSystemNote(note)
		}
	}
}

// View renders the current state.
//
// Layout (three-pane shell):
//
//	┌────────────┬─────────────────────────┬──────────────┐
//	│ workspaces │  transcript             │  session     │
//	│  (stub)    │                         │  info        │
//	│            ├─────────────────────────┤              │
//	│            │  composer / help-sheet  │              │
//	└────────────┴─────────────────────────┴──────────────┘
//
// The right pane carries everything that used to live in the bottom status
// bar (model, mode, context %, cost, rate-limit). The hint footer is gone —
// `?` toggles a bottom-sheet help panel that replaces the composer area.
func (a *App) View() tea.View {
	if a.quitting {
		v := tea.NewView("")
		v.AltScreen = true
		return v
	}
	if a.width == 0 || a.height == 0 {
		v := tea.NewView("starting…")
		v.AltScreen = true
		return v
	}

	leftW, centerW, rightW := paneLayout(a.width)

	// Inset chat content by chatPadH cells on each side so it doesn't sit
	// flush against the side panels' rounded borders.
	const chatPadH = 2
	innerCenterW := centerW - 2*chatPadH
	if innerCenterW < 8 {
		innerCenterW = 8
	}

	// Composer text width: 2 cells for the "> " prompt, the rest for input.
	composerInnerW := innerCenterW - 2
	if composerInnerW < 4 {
		composerInnerW = 4
	}
	composerH := a.composer.LineCount()
	if composerH < 1 {
		composerH = 1
	}
	if maxH := a.height / 3; composerH > maxH && maxH > 1 {
		composerH = maxH
	}

	// Bottom row: a full-width `─` rule (with worktree/branch label) followed
	// by either the composer or the help sheet. The rule spans the entire
	// chat column; the content beneath is inset by chatPadH on each side.
	rule := a.composerRule(centerW)
	var bottomBody string
	var bottomH int
	if a.showHelp {
		sheetH := len(DefaultHelp())/3 + 4 // -1 because rule is rendered separately
		if sheetH > a.height/2 {
			sheetH = a.height / 2
		}
		bottomBody = indentLines(renderHelpSheet(a.theme, innerCenterW, sheetH), chatPadH)
		bottomH = 1 + sheetH
	} else {
		const hintText = "? for help"
		hintLen := len(hintText) + 1 // 1 col separator before the hint
		taW := composerInnerW - hintLen
		if taW < 4 {
			taW = 4
		}
		a.composer.SetWidth(taW)
		a.composer.SetHeight(composerH)
		taLines := strings.Split(a.composer.View(), "\n")
		hint := a.theme.Dim.Render(hintText)
		composerLines := make([]string, len(taLines))
		for i, l := range taLines {
			prefix := "  "
			if i == 0 {
				prefix = a.theme.Fg.Render("> ")
			}
			line := prefix + l
			pad := innerCenterW - lipgloss.Width(line) - lipgloss.Width(hint)
			if pad < 1 {
				pad = 1
			}
			if i == 0 {
				line += strings.Repeat(" ", pad) + hint
			}
			composerLines[i] = line
		}
		bottomBody = indentLines(strings.Join(composerLines, "\n"), chatPadH)
		bottomH = 1 + composerH
	}
	bottom := rule + "\n" + bottomBody

	// Center column: transcript + bottom (composer/sheet). No header, no
	// background fill — the chat column is just terminal default. Reserve
	// one trailing row so the composer doesn't sit on the same baseline as
	// the side panels' bottom border (visually misaligned otherwise).
	const chatBottomPad = 1
	transcriptH := a.height - bottomH - chatBottomPad
	if transcriptH < 3 {
		transcriptH = 3
	}

	if a.viewport.Height() != transcriptH || a.viewport.Width() != innerCenterW {
		a.viewport.SetWidth(innerCenterW)
		a.viewport.SetHeight(transcriptH)
	}
	wasAtBottom := a.viewport.AtBottom()
	tc := a.transcript.Render(a.theme, innerCenterW)
	// Bottom-anchor the transcript so the welcome note and the first few
	// messages sit just above the composer rather than floating at the top
	// of an empty viewport.
	if lines := strings.Count(tc, "\n") + 1; lines < transcriptH {
		tc = strings.Repeat("\n", transcriptH-lines) + tc
	}
	a.viewport.SetContent(tc)
	if wasAtBottom {
		a.viewport.GotoBottom()
	}
	tx := strings.TrimRight(a.viewport.View(), "\n")
	if !a.viewport.AtBottom() {
		txLines := strings.Split(tx, "\n")
		if len(txLines) > 0 {
			txLines[len(txLines)-1] = a.theme.Accent.Render("↓ jump to bottom")
		}
		tx = strings.Join(txLines, "\n")
	}
	tx = indentLines(tx, chatPadH)

	center := lipgloss.JoinVertical(lipgloss.Left, tx, bottom)

	// Panes sit flush against the chat column — the colour contrast IS the
	// divider, so no whitespace gutter is drawn.
	var content string
	if leftW == 0 && rightW == 0 {
		content = center
	} else {
		left := renderLeftPane(a.theme, leftW, a.height)
		right := renderRightPane(a, rightW, a.height)
		content = lipgloss.JoinHorizontal(lipgloss.Top, left, center, right)
	}

	v := tea.NewView(content)
	v.AltScreen = true
	v.MouseMode = tea.MouseModeCellMotion
	return v
}
