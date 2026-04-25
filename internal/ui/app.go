package ui

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/bubbles/v2/textarea"
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
	composer    Composer
	status      StatusBar
	help        []HelpRow
	sessionUUID string
	firstTurn   bool

	// One driver lives per turn.
	driver *claude.Driver
	cancel context.CancelFunc

	// Aggregated session usage (running totals).
	tokens    int
	ctxWindow int
	costUSD   float64
	model     string
	permMode  string
	rateState string

	// Banner text for the session (after first system/init).
	banner string

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
		composer:    NewComposer(),
		help:        DefaultHelp(),
		sessionUUID: uuid.NewString(),
		firstTurn:   true,
		ctxWindow:   200000, // sane default until the first `result` updates it
		debug:       debug,
		jflowVer:    version,
	}
	a.openLog()
	return a
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
	note := "new session " + a.sessionUUID[:8] + " — type a message and press enter"
	if a.logPath != "" {
		note += "  ·  log: " + a.logPath
	}
	a.transcript.AddSystemNote(note)
	return textarea.Blink
}

// driverEventMsg wraps a claude event so it can be delivered as a tea.Msg.
type driverEventMsg struct{ ev claude.Event }

// driverDoneMsg is delivered when the event channel closes (after DriverExit).
type driverDoneMsg struct{}

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
		a.composer.SetWidth(m.Width - 2)
		return a, nil

	case tea.KeyPressMsg:
		key := m.String()
		switch key {
		case "ctrl+c", "esc":
			if a.driver != nil {
				_ = a.driver.Interrupt()
				if a.debug {
					a.writeMeta("key", map[string]any{"key": key, "context": "interrupt"})
				}
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
		case "ctrl+x":
			if a.driver != nil {
				_ = a.driver.Interrupt()
			}
			return a, nil
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
			if a.driver != nil {
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
		a.status.StatusWord = ""
		return a, nil
	}

	// Pass remaining messages to the composer.
	var cmd tea.Cmd
	a.composer, cmd = a.composer.Update(msg)
	return a, cmd
}

// send spawns a new claude turn with the given user message.
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
	d, err := claude.Spawn(ctx, opts)
	if err != nil {
		a.transcript.AddSystemNote("spawn error: " + err.Error())
		a.writeMeta("spawn_error", map[string]any{"err": err.Error()})
		return nil
	}
	a.driver = d
	a.writeMeta("spawn", map[string]any{
		"session_uuid": a.sessionUUID,
		"argv":         d.Argv(),
		"prompt":       text,
	})
	a.status.StatusWord = "thinking…"
	return readDriver(d)
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
		if e.IsError {
			a.transcript.AddSystemNote("error: " + e.Result + " (terminal_reason=" + e.TerminalReason + ")")
		}

	case claude.ParseError:
		a.transcript.AddSystemNote("parse error: " + e.Err.Error())

	case claude.DriverExit:
		if e.Err != nil {
			a.transcript.AddSystemNote("subprocess exited: " + e.Err.Error())
		}
	}
}

// View renders the current state.
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

	statusH := 1
	composerH := 5 // textarea height + borders
	transcriptH := a.height - statusH - composerH
	if transcriptH < 5 {
		transcriptH = 5
	}

	tx := a.transcript.Render(a.theme, a.width-2)
	tx = clipToHeight(tx, transcriptH)

	help := a.renderHelp()
	statusLine := a.status.View(a.theme, a.width)

	// Single visible "> " cue rendered once, then the composer (which has
	// Prompt="") so we don't get a stack of prompts on multi-line input.
	composerLines := strings.Split(a.composer.View(), "\n")
	for i, l := range composerLines {
		if i == 0 {
			composerLines[i] = a.theme.Accent.Render("> ") + l
		} else {
			composerLines[i] = "  " + l
		}
	}
	composerView := strings.Join(composerLines, "\n")

	content := strings.Join([]string{
		tx,
		statusLine,
		composerView,
		help,
	}, "\n")
	v := tea.NewView(content)
	v.AltScreen = true
	return v
}

func (a *App) renderHelp() string {
	parts := make([]string, 0, len(a.help))
	for _, h := range a.help {
		parts = append(parts, a.theme.HelpKey.Render(h.Key)+" "+a.theme.HelpDesc.Render(h.Desc))
	}
	return strings.Join(parts, "  ")
}

// clipToHeight returns the last n lines of s, padding with blanks if shorter.
func clipToHeight(s string, n int) string {
	lines := strings.Split(s, "\n")
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	for len(lines) < n {
		lines = append([]string{""}, lines...)
	}
	return strings.Join(lines, "\n")
}
