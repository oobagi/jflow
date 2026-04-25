package ui

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"charm.land/bubbles/v2/textarea"
	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/oobagi/jflow/internal/claude"
	"github.com/oobagi/jflow/internal/config"
	"github.com/oobagi/jflow/internal/session"
	"github.com/oobagi/jflow/internal/state"
	"github.com/oobagi/jflow/internal/workspace"
)

// focusKind identifies which pane currently has keyboard focus. Tab cycles
// the focus left → composer → right → left, mirroring how the three columns
// sit visually on screen.
type focusKind int

const (
	focusComposer focusKind = iota
	focusLeft
	focusRight
)

// App is the root Bubble Tea model. It manages a list of workspaces (folders)
// each containing zero or more sessions (independent claude conversations).
// Only one session is "active" at a time — its transcript fills the chat
// column. Without an active session the chat area shows a placeholder.
type App struct {
	theme      Theme
	width      int
	height     int
	transcript Transcript
	viewport   viewport.Model
	composer   Composer
	status     StatusBar

	// Active session — drives chat. Empty = empty state, no chat.
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

	// Worktree label shown inline with the composer rule. Refreshed per
	// session activation: general sessions show prefs.DefaultDir, workspace
	// sessions show their workspace's cwd. branch is recomputed by running
	// `git rev-parse --abbrev-ref HEAD` inside that cwd.
	worktree string
	branch   string
	// activeCWD is the absolute path the active session's claude subprocess
	// runs in. Empty until a session is activated.
	activeCWD string

	// showHelp toggles the full-screen help overlay (#26).
	showHelp bool

	quitting bool

	// Debug logging: always-on raw JSONL log + jflow meta entries.
	// `--debug` adds extra meta entries (key presses, etc.).
	debug       bool
	logFile     *os.File
	logPath     string
	jflowVer    string
	workspaceID string

	// Left pane state — a file-browser-style tree where workspaces are folders
	// (collapsible) and sessions are leaves. focusLeft swaps focus between the
	// composer and the tree (Tab toggles). expanded tracks which workspaces
	// are open; treeCursor is the index into the flattened render rows.
	wsStore        *workspace.Store
	sessStore      *session.Store
	prefs          config.Preferences
	wsList         []workspace.Workspace
	expanded       map[string]bool
	focus          focusKind
	treeCursor     int
	treeConfirmDel bool
	treePendingID  string
	treePendingKnd string // "ws" | "sess"

	// Workspace add prompt overlay. Open when user presses `a` while the tree
	// has focus. Single-line textinput pre-filled with the launch cwd.
	wsPromptOpen  bool
	wsPromptInput textinput.Model

	// In-memory history of user sends for ↑/↓ recall (#29). Current-session
	// only — cross-session recall is out of scope for v0. Pointer semantics:
	//   histIdx == len(history)  → not recalling (composer reflects user input)
	//   histIdx <  len(history)  → recalling history[histIdx]
	// Any keypress that mutates the composer (i.e. typing) resets histIdx
	// back to len(history) so the next ↑ starts from the most-recent send.
	history []string
	histIdx int
}

// NewApp constructs the App with no active session — the user picks or
// creates one via the left pane (⌃W workspaces, ⌃S sessions).
//
// `debug` enables verbose meta entries in the session log (key events, etc.).
// `version` is recorded in each session-start meta entry.
// `wsStore`/`sessStore` are the persistent stores (may be nil on bootstrap
// failure — the UI degrades to a read-only/empty pane).
// `workspaceID` is the workspace bootstrapped for the launch cwd (#32).
//
// Session logs are written to ~/.jflow/state/logs/<ts>-<sid8>.jsonl on
// session activation; `last.jsonl` is updated on every activation so the
// most recent session is always reachable.
func NewApp(debug bool, version string, wsStore *workspace.Store, sessStore *session.Store, prefs config.Preferences, workspaceID string) *App {
	a := &App{
		theme:       DefaultTheme(),
		viewport:    viewport.New(),
		composer:    NewComposer(),
		firstTurn:   true,
		ctxWindow:   200000,
		debug:       debug,
		jflowVer:    version,
		workspaceID: workspaceID,
		wsStore:     wsStore,
		sessStore:   sessStore,
		prefs:       prefs,
		expanded:    map[string]bool{},
	}
	a.viewport.SoftWrap = true
	a.worktree, a.branch = detectWorktreeBranch("")
	a.wsRefresh()
	// Auto-expand the launch-cwd workspace so the user sees its sessions
	// (or "(none)") immediately on startup without an extra keypress.
	if workspaceID != "" {
		a.expanded[workspaceID] = true
	}
	// Always have an active session: resume the most recent one in the
	// launch workspace, or create a fresh "default" session if none exist.
	// This way the center pane is always a usable chat without forcing the
	// user through workspace/session bookkeeping first.
	a.ensureActiveSession()
	return a
}

// helpForFocus returns the keybind set relevant to the currently-focused
// pane. The right pane has no actions wired so we show its placeholder list.
func (a *App) helpForFocus() []HelpRow {
	switch a.focus {
	case focusLeft:
		return TreeHelp()
	case focusRight:
		return RightHelp()
	default:
		return ChatHelp()
	}
}

// cycleFocus returns the next focus state in the composer → left → right
// ring (Tab forward) or its reverse (Shift+Tab). Tabbing out of the composer
// lands on the tree first because that's the most common next action; the
// right pane is informational and comes second.
func (a *App) cycleFocus(dir int) focusKind {
	switch a.focus {
	case focusComposer:
		if dir > 0 {
			return focusLeft
		}
		return focusRight
	case focusLeft:
		if dir > 0 {
			return focusRight
		}
		return focusComposer
	case focusRight:
		if dir > 0 {
			return focusComposer
		}
		return focusLeft
	}
	return focusComposer
}

// ensureActiveSession guarantees the chat has something to show. Preference
// order: most-recent general session → most-recent session in the active
// workspace → newly-created general session. The launch workspace is always
// kept alive (recreated for cwd if it was deleted) so the tree still shows
// it, but the chat default lives outside any workspace.
func (a *App) ensureActiveSession() {
	if a.sessionUUID != "" {
		return
	}
	if a.sessStore == nil || a.wsStore == nil {
		return
	}
	if a.workspaceID != "" {
		if _, err := a.wsStore.Get(a.workspaceID); err != nil {
			a.workspaceID = ""
		}
	}
	if a.workspaceID == "" {
		dir := a.prefs.ResolveDefaultDir()
		if dir == "" {
			if cwd, err := os.Getwd(); err == nil {
				dir = cwd
			}
		}
		if dir != "" {
			if w, _, err := a.wsStore.EnsureForCWD(dir); err == nil {
				a.workspaceID = w.ID
				a.expanded[w.ID] = true
				a.wsRefresh()
			}
		}
	}
	var pick *session.Session
	for _, s := range a.sessStore.List() {
		if s.WorkspaceID != "" {
			continue
		}
		if pick == nil || s.LastUsedAt.After(pick.LastUsedAt) {
			cp := s
			pick = &cp
		}
	}
	if pick == nil && a.workspaceID != "" {
		for _, s := range a.sessStore.ListByWorkspace(a.workspaceID) {
			if pick == nil || s.LastUsedAt.After(pick.LastUsedAt) {
				cp := s
				pick = &cp
			}
		}
	}
	if pick != nil {
		a.activateSession(pick.ID)
		return
	}
	s := session.New("", "session 1")
	if err := a.sessStore.Add(s); err != nil {
		return
	}
	a.activateSession(s.ID)
}

// wsRefresh reloads the workspace list (LastUsedAt desc, launch-cwd pinned
// on top) and clamps treeCursor.
func (a *App) wsRefresh() {
	if a.wsStore == nil {
		a.wsList = nil
		a.treeCursor = 0
		return
	}
	list := a.wsStore.List()
	for i := 0; i < len(list); i++ {
		for j := i + 1; j < len(list); j++ {
			if list[j].LastUsedAt.After(list[i].LastUsedAt) {
				list[i], list[j] = list[j], list[i]
			}
		}
	}
	if a.workspaceID != "" {
		for i, w := range list {
			if w.ID == a.workspaceID && i != 0 {
				active := list[i]
				list = append(list[:i], list[i+1:]...)
				list = append([]workspace.Workspace{active}, list...)
				break
			}
		}
	}
	a.wsList = list
	rows := a.tree()
	if a.treeCursor >= len(rows) {
		a.treeCursor = len(rows) - 1
	}
	if a.treeCursor < 0 {
		a.treeCursor = 0
	}
}

// treeRow is one rendered line in the tree pane. Kinds:
//   - "action":  the "+ new session" row pinned at the top
//   - "general": a session not bound to any workspace (lives in the
//                default directory)
//   - "ws":      a workspace folder (▸ collapsed / ▾ expanded)
//   - "sess":    a session indented inside an expanded workspace
type treeRow struct {
	kind        string
	workspaceID string
	sessionID   string
	wsName      string
	sessName    string
	wsExpanded  bool
	sessCount   int
}

// tree flattens the model into rendering rows. Order is: action row, general
// sessions, then workspaces (with their nested sessions when expanded).
func (a *App) tree() []treeRow {
	var rows []treeRow
	rows = append(rows, treeRow{kind: "action"})
	if a.sessStore != nil {
		for _, s := range a.sessStore.List() {
			if s.WorkspaceID == "" {
				rows = append(rows, treeRow{
					kind:      "general",
					sessionID: s.ID,
					sessName:  s.Name,
				})
			}
		}
	}
	for _, w := range a.wsList {
		count := 0
		if a.sessStore != nil {
			count = len(a.sessStore.ListByWorkspace(w.ID))
		}
		rows = append(rows, treeRow{
			kind:        "ws",
			workspaceID: w.ID,
			wsName:      w.Name,
			wsExpanded:  a.expanded[w.ID],
			sessCount:   count,
		})
		if a.expanded[w.ID] && a.sessStore != nil {
			// Static order: sessions render in store insertion order so the
			// list doesn't reshuffle as you switch between them.
			for _, s := range a.sessStore.ListByWorkspace(w.ID) {
				rows = append(rows, treeRow{
					kind:        "sess",
					workspaceID: w.ID,
					sessionID:   s.ID,
					sessName:    s.Name,
				})
			}
		}
	}
	return rows
}

// detectWorktreeBranch returns a home-shortened display path for cwd and the
// current git branch. cwd is the directory to inspect; pass the active
// session's effective cwd to keep the composer rule in sync. Empty cwd
// falls back to os.Getwd().
func detectWorktreeBranch(cwd string) (string, string) {
	if cwd == "" {
		c, err := os.Getwd()
		if err != nil {
			return "", ""
		}
		cwd = c
	}
	display := cwd
	if home, err := os.UserHomeDir(); err == nil {
		if cwd == home {
			display = "~"
		} else if strings.HasPrefix(cwd, home+string(os.PathSeparator)) {
			display = "~" + cwd[len(home):]
		}
	}
	branch := ""
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	cmd.Dir = cwd
	if out, err := cmd.Output(); err == nil {
		b := strings.TrimSpace(string(out))
		if b != "" && b != "HEAD" {
			branch = b
		}
	}
	return display, branch
}

// sessionCWD resolves the directory a session's claude subprocess should run
// in. Workspace sessions use their workspace's cwd; general sessions use the
// configured default_dir (or, if unset/unresolvable, the launch cwd).
func (a *App) sessionCWD(s session.Session) string {
	if s.WorkspaceID != "" && a.wsStore != nil {
		if w, err := a.wsStore.Get(s.WorkspaceID); err == nil {
			return w.CWD
		}
	}
	if dir := a.prefs.ResolveDefaultDir(); dir != "" {
		return dir
	}
	if cwd, err := os.Getwd(); err == nil {
		return cwd
	}
	return ""
}

// openLog opens the per-session log file in append mode. The filename is
// stable across activations (`<sessionUUID>.jsonl`) so the full conversation
// history accumulates in one place — see hydrateFromLog for replay.
//
// Closes any previously-open log first. Failures surface as a transcript note.
func (a *App) openLog() {
	if a.sessionUUID == "" {
		return
	}
	if a.logFile != nil {
		_ = a.logFile.Close()
		a.logFile = nil
	}
	dir, err := state.LogsDir()
	if err != nil {
		a.transcript.AddSystemNote("log dir unavailable: " + err.Error())
		return
	}
	path := filepath.Join(dir, a.sessionUUID+".jsonl")
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0o644)
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
		"workspace_id":  a.workspaceID,
	})
}

// hydrateFromLog reads the per-session JSONL log and replays it so the
// transcript reflects prior turns. Returns true if at least one claude
// `result` event was replayed — the signal that claude has actually driven
// this session before, which the caller uses to decide between --session-id
// and --resume on the next spawn. Best-effort: per-line parse errors are
// swallowed.
func (a *App) hydrateFromLog() bool {
	if a.logPath == "" {
		return false
	}
	data, err := os.ReadFile(a.logPath)
	if err != nil {
		return false
	}
	driven := false
	for _, line := range bytes.Split(data, []byte{'\n'}) {
		if len(bytes.TrimSpace(line)) == 0 {
			continue
		}
		var meta struct {
			Jflow string `json:"_jflow"`
			Text  string `json:"text"`
		}
		if err := json.Unmarshal(line, &meta); err == nil && meta.Jflow != "" {
			if meta.Jflow == "user_send" && meta.Text != "" {
				a.transcript.AddUserMessage(meta.Text)
			}
			continue
		}
		ev, err := claude.ParseLine(line)
		if err != nil || ev == nil {
			continue
		}
		switch e := ev.(type) {
		case claude.DriverExit, claude.ParseError:
			continue
		case claude.Result:
			// Only count successful turns: an error result means claude
			// rejected the spawn (e.g. "no conversation found"), which
			// must NOT mark the session as driven — that'd lock us into
			// --resume forever and the bug repeats.
			if !e.IsError {
				driven = true
			}
		}
		a.applyEvent(ev)
	}
	return driven
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
		// Resize the chat viewport eagerly so its internal scroll model
		// matches the new dimensions even before View() is next called.
		// (View() will overwrite Width/Height again, but the Render done
		// during this Update tick uses these values.)
		_, centerW, _ := paneLayout(a.width)
		innerW := centerW - 4
		if innerW < 8 {
			innerW = 8
		}
		a.viewport.SetWidth(innerW)
		if a.height > 8 {
			a.viewport.SetHeight(a.height - 6)
		}
		if a.debug {
			a.writeMeta("resize", map[string]any{"w": a.width, "h": a.height})
		}
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
		// Help is a non-modal cheatsheet that pops up under the composer:
		// `?` and `esc` close it, but every other key keeps flowing through
		// to the composer so the user can keep typing while it's open.
		if a.showHelp {
			switch key {
			case "?", "esc":
				a.showHelp = false
				if a.debug {
					a.writeMeta("key", map[string]any{"key": key, "context": "help_close"})
				}
				return a, nil
			}
		} else if key == "?" && a.composer.IsEmpty() {
			// Open help only when the composer is empty so the user can still
			// type "?" in a message.
			a.showHelp = true
			if a.debug {
				a.writeMeta("key", map[string]any{"key": key, "context": "help_open"})
			}
			return a, nil
		}
		// Workspace add prompt overlay (textinput) takes priority — every
		// keystroke flows through it until ⏎ confirms or esc cancels.
		if a.wsPromptOpen {
			cmd := a.handleWorkspacePromptKey(m)
			return a, cmd
		}
		// Tab cycles focus left → composer → right → left. Shift+Tab walks
		// the same ring backwards. Always intercept so the textarea never
		// sees the key (otherwise shift+tab would type a literal).
		if key == "tab" {
			a.focus = a.cycleFocus(+1)
			if a.focus == focusLeft {
				a.wsRefresh()
			}
			return a, nil
		}
		if key == "shift+tab" {
			a.focus = a.cycleFocus(-1)
			if a.focus == focusLeft {
				a.wsRefresh()
			}
			return a, nil
		}
		// While the tree pane has focus it owns navigation/edit keys; the
		// composer is bypassed entirely.
		if a.focus == focusLeft {
			if cmd, handled := a.handleTreeKey(key); handled {
				return a, cmd
			}
		} else if a.focus == focusRight {
			// Right pane has no actions yet — only Tab/Shift+Tab/esc move
			// out, everything else is a no-op so the composer doesn't catch
			// stray typing.
			switch key {
			case "esc":
				a.focus = focusComposer
				return a, nil
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
			// While in history recall (#29), esc first clears the recalled
			// text and exits recall mode rather than quitting — a second
			// esc on an empty/idle composer then falls through to quit.
			if a.recalling() {
				a.histIdx = len(a.history)
				a.composer.Reset()
				if a.debug {
					a.writeMeta("key", map[string]any{"key": key, "context": "history_cancel"})
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
		case "ctrl+k":
			// compact: route through dispatch so the user sees a "compacting…"
			// note in the transcript instead of just a bare "/compact" send.
			return a, a.dispatch("/compact")
		case "shift+enter", "ctrl+j":
			// soft newline — forward to the textarea as a synthetic Enter
			cmd := a.composer.InsertNewline()
			if a.debug {
				a.writeMeta("key", map[string]any{"key": key, "context": "newline"})
			}
			return a, cmd
		case "enter":
			if a.sessionUUID == "" {
				// No active session — composer should be hidden, but if a
				// stray enter arrives just point the user at session create.
				return a, nil
			}
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
			return a, a.dispatch(text)
		case "up":
			// History recall (#29). Only intercept when the composer is
			// empty (nothing to lose) or when we're already navigating
			// recall (so successive ↑ walk further back). Otherwise fall
			// through so the textarea moves the cursor up between lines.
			// While recalling, consume the keypress even if we're already
			// at the oldest entry — otherwise the typing-detection branch
			// below would reset histIdx and silently exit recall mode
			// while leaving the recalled text in the composer, which then
			// breaks the next ↓.
			if a.recalling() {
				a.recallPrev()
				if a.debug {
					a.writeMeta("key", map[string]any{"key": key, "context": "history_prev"})
				}
				return a, nil
			}
			if a.composer.IsEmpty() {
				if a.recallPrev() {
					if a.debug {
						a.writeMeta("key", map[string]any{"key": key, "context": "history_prev"})
					}
					return a, nil
				}
			}
		case "down":
			if a.recalling() {
				a.recallNext()
				if a.debug {
					a.writeMeta("key", map[string]any{"key": key, "context": "history_next"})
				}
				return a, nil
			}
		}
		// Any other keypress that reaches the composer counts as "typing"
		// and resets the recall pointer so the next ↑ starts fresh from
		// the most-recent send (#29).
		a.histIdx = len(a.history)
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

// dispatch is the entry point for any composer submission (enter or a wired
// keybind like ⌃K). It intercepts harness-level slash commands so they don't
// silently fall through to claude as opaque text — claude doesn't recognise
// `/new` at all, and `/compact` runs but its result event is easy to miss
// in the stream — and surfaces visible feedback in the transcript before
// either resetting the session locally or forwarding to claude.
//
// Returns nil for fully-local commands (no claude turn was queued).
func (a *App) dispatch(text string) tea.Cmd {
	switch strings.TrimSpace(text) {
	case "/new", "/clear":
		// Local reset: spin up a fresh session in the same workspace as the
		// current one so the user keeps their cwd context. Falls back to a
		// general session when there's no active workspace.
		a.startFreshSession()
		return nil
	case "/compact":
		// Claude handles /compact natively, but the only signal the user
		// gets back is a quiet result event — so prefix a system note so the
		// chat panel makes it obvious something kicked off.
		a.transcript.AddSystemNote("compacting context…")
		return a.send(text)
	}
	return a.send(text)
}

// startFreshSession is the local handler for `/new` and `/clear`. Refuses
// while a turn is in flight (matches activateSession's policy) and reuses
// the existing create helpers so naming, expansion, and tree-cursor
// placement stay consistent with creating a session via the tree pane.
func (a *App) startFreshSession() {
	if a.driver != nil || a.spawning {
		a.transcript.AddSystemNote("can't start a new session while a turn is running — cancel first (⌃C)")
		return
	}
	if a.sessStore == nil {
		return
	}
	var ws string
	if a.sessionUUID != "" {
		if s, err := a.sessStore.Get(a.sessionUUID); err == nil {
			ws = s.WorkspaceID
		}
	}
	if ws == "" {
		a.createGeneralSession()
	} else {
		a.createSessionIn(ws)
	}
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
	// Push onto the in-memory history ring and reset the recall pointer
	// (#29). De-dup against the most-recent entry so repeated identical
	// sends don't bloat the ring.
	if n := len(a.history); n == 0 || a.history[n-1] != text {
		a.history = append(a.history, text)
	}
	a.histIdx = len(a.history)
	ctx, cancel := context.WithCancel(context.Background())
	a.cancel = cancel
	opts := claude.SpawnOpts{
		SessionID: a.sessionUUID,
		Resume:    !a.firstTurn,
		Prompt:    text,
		CWD:       a.activeCWD,
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
//
//	── ~/.jflow · main ──────────────────
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

// recalling returns true when the composer currently displays a message
// pulled from the history ring (i.e. the user is mid-recall).
func (a *App) recalling() bool {
	return a.histIdx < len(a.history)
}

// recallPrev walks one step back in history and loads that message into
// the composer. Returns false when there's nothing older to show (so the
// caller can fall through to the default keypress handling).
func (a *App) recallPrev() bool {
	if len(a.history) == 0 || a.histIdx == 0 {
		return false
	}
	a.histIdx--
	a.composer.SetValue(a.history[a.histIdx])
	return true
}

// recallNext walks one step forward in history. Stepping past the newest
// entry empties the composer and exits recall mode (histIdx == len).
func (a *App) recallNext() {
	if !a.recalling() {
		return
	}
	a.histIdx++
	if a.histIdx >= len(a.history) {
		a.histIdx = len(a.history)
		a.composer.Reset()
		return
	}
	a.composer.SetValue(a.history[a.histIdx])
}

// handleTreeKey processes keypresses while the tree pane is focused.
// j/k/↑/↓: move cursor · l/→: expand workspace · h/←: collapse (or jump up
// to parent on a session) · ⏎: toggle workspace expansion / activate
// session · n: new session in workspace (creates the workspace's first
// session if needed) · a: add a new workspace via path prompt · x: delete
// with y/N confirm · tab/esc: blur back to composer.
func (a *App) handleTreeKey(key string) (tea.Cmd, bool) {
	if a.wsStore == nil {
		a.focus = focusComposer
		return nil, false
	}
	rows := a.tree()

	// Delete confirmation gates everything else — y/Y removes, n/N/esc bails.
	if a.treeConfirmDel {
		switch key {
		case "y", "Y":
			id, kind := a.treePendingID, a.treePendingKnd
			a.treeConfirmDel = false
			a.treePendingID = ""
			a.treePendingKnd = ""
			a.deleteTreeNode(id, kind)
			a.wsRefresh()
			// If we just deleted the active session, fall back to the most
			// recent remaining session in the launch workspace (or create a
			// fresh "default") so the chat stays available.
			if a.sessionUUID == "" {
				a.ensureActiveSession()
			}
			return nil, true
		case "n", "N", "esc":
			a.treeConfirmDel = false
			a.treePendingID = ""
			a.treePendingKnd = ""
			return nil, true
		}
		return nil, true
	}

	switch key {
	case "tab":
		a.focus = a.cycleFocus(+1)
		return nil, true
	case "shift+tab":
		a.focus = a.cycleFocus(-1)
		return nil, true
	case "esc":
		a.focus = focusComposer
		return nil, true
	case "up", "k":
		if a.treeCursor > 0 {
			a.treeCursor--
		}
		return nil, true
	case "down", "j":
		if a.treeCursor < len(rows)-1 {
			a.treeCursor++
		}
		return nil, true
	case "right", "l":
		if cur, ok := a.curRow(rows); ok && cur.kind == "ws" && !cur.wsExpanded {
			a.expanded[cur.workspaceID] = true
		}
		return nil, true
	case "left", "h":
		if cur, ok := a.curRow(rows); ok {
			if cur.kind == "ws" && cur.wsExpanded {
				a.expanded[cur.workspaceID] = false
			} else if cur.kind == "sess" {
				for i := a.treeCursor - 1; i >= 0; i-- {
					if rows[i].kind == "ws" && rows[i].workspaceID == cur.workspaceID {
						a.treeCursor = i
						break
					}
				}
			}
		}
		return nil, true
	case "enter":
		if cur, ok := a.curRow(rows); ok {
			switch cur.kind {
			case "action":
				a.createGeneralSession()
			case "ws":
				a.expanded[cur.workspaceID] = !a.expanded[cur.workspaceID]
			case "general", "sess":
				a.activateSession(cur.sessionID)
			}
		}
		return nil, true
	case "a":
		a.openWorkspacePrompt()
		return textinput.Blink, true
	case "n":
		// `n` on the action row or any general session creates another
		// general session (default-dir, no workspace). On a workspace or
		// workspace session, it creates a session in that workspace.
		cur, ok := a.curRow(rows)
		if !ok || cur.kind == "action" || cur.kind == "general" {
			a.createGeneralSession()
			return nil, true
		}
		a.createSessionIn(cur.workspaceID)
		return nil, true
	case "x":
		cur, ok := a.curRow(rows)
		if !ok || cur.kind == "action" {
			return nil, true
		}
		a.treeConfirmDel = true
		if cur.kind == "ws" {
			a.treePendingID = cur.workspaceID
			a.treePendingKnd = "ws"
		} else {
			a.treePendingID = cur.sessionID
			a.treePendingKnd = "sess"
		}
		return nil, true
	}
	return nil, true
}

// curRow returns the tree row under the cursor, or false if the tree is
// empty / cursor is out of range.
func (a *App) curRow(rows []treeRow) (treeRow, bool) {
	if a.treeCursor < 0 || a.treeCursor >= len(rows) {
		return treeRow{}, false
	}
	return rows[a.treeCursor], true
}

// createGeneralSession adds a session not bound to any workspace — a "default
// directory" chat. Activates it and parks the cursor on the new row.
func (a *App) createGeneralSession() {
	if a.sessStore == nil {
		return
	}
	count := 0
	for _, s := range a.sessStore.List() {
		if s.WorkspaceID == "" {
			count++
		}
	}
	name := fmt.Sprintf("session %d", count+1)
	s := session.New("", name)
	if err := a.sessStore.Add(s); err != nil {
		a.transcript.AddSystemNote("session add failed: " + err.Error())
		return
	}
	a.activateSession(s.ID)
	for i, r := range a.tree() {
		if r.kind == "general" && r.sessionID == s.ID {
			a.treeCursor = i
			break
		}
	}
	if a.debug {
		a.writeMeta("session_create", map[string]any{"id": s.ID, "name": s.Name, "general": true})
	}
}

// createSessionIn adds a new session under the given workspace, expands it,
// activates the session, and refreshes the cursor onto the new row.
func (a *App) createSessionIn(workspaceID string) {
	if a.sessStore == nil || workspaceID == "" {
		return
	}
	count := len(a.sessStore.ListByWorkspace(workspaceID))
	name := fmt.Sprintf("session %d", count+1)
	s := session.New(workspaceID, name)
	if err := a.sessStore.Add(s); err != nil {
		a.transcript.AddSystemNote("session add failed: " + err.Error())
		return
	}
	a.expanded[workspaceID] = true
	a.activateSession(s.ID)
	// Move the tree cursor to the new session row so x/⏎ act on it next.
	rows := a.tree()
	for i, r := range rows {
		if r.kind == "sess" && r.sessionID == s.ID {
			a.treeCursor = i
			break
		}
	}
	if a.debug {
		a.writeMeta("session_create", map[string]any{"id": s.ID, "name": s.Name, "workspace_id": workspaceID})
	}
}

// deleteTreeNode removes a workspace (cascading its sessions) or a single
// session and clears chat state if the active session is touched.
func (a *App) deleteTreeNode(id, kind string) {
	switch kind {
	case "ws":
		if a.sessStore != nil {
			for _, s := range a.sessStore.ListByWorkspace(id) {
				_ = a.sessStore.Remove(s.ID)
				if s.ID == a.sessionUUID {
					a.deactivateSession()
				}
			}
		}
		if err := a.wsStore.Remove(id); err != nil {
			a.transcript.AddSystemNote("workspace remove failed: " + err.Error())
		}
		delete(a.expanded, id)
		if a.debug {
			a.writeMeta("ws_remove", map[string]any{"id": id})
		}
	case "sess":
		if a.sessStore == nil {
			return
		}
		if err := a.sessStore.Remove(id); err != nil {
			a.transcript.AddSystemNote("session remove failed: " + err.Error())
			return
		}
		if id == a.sessionUUID {
			a.deactivateSession()
		}
		if a.debug {
			a.writeMeta("session_remove", map[string]any{"id": id})
		}
	}
}

// openWorkspacePrompt initialises the path textinput with the launch cwd and
// makes the bottom-sheet overlay visible. Caller is expected to also return
// textinput.Blink so the cursor renders.
func (a *App) openWorkspacePrompt() {
	cwd, _ := os.Getwd()
	ti := textinput.New()
	ti.Placeholder = "/path/to/folder"
	ti.SetValue(cwd)
	ti.CharLimit = 0
	ti.SetWidth(40)
	// Drop the default "> " prompt — the framed title + box already cue the
	// user that this is an input, so the prompt would just look like noise
	// inside the rounded box.
	ti.Prompt = ""
	ti.Focus()
	a.wsPromptInput = ti
	a.wsPromptOpen = true
}

// handleWorkspacePromptKey runs while the path prompt is open. ⏎ confirms,
// esc cancels, everything else flows into the textinput.
func (a *App) handleWorkspacePromptKey(msg tea.KeyPressMsg) tea.Cmd {
	switch msg.String() {
	case "esc":
		a.wsPromptOpen = false
		return nil
	case "enter":
		raw := strings.TrimSpace(a.wsPromptInput.Value())
		a.wsPromptOpen = false
		if raw == "" {
			return nil
		}
		path, err := filepath.Abs(expandHome(raw))
		if err != nil {
			a.transcript.AddSystemNote("workspace add: " + err.Error())
			return nil
		}
		w, created, err := a.wsStore.EnsureForCWD(path)
		if err != nil {
			a.transcript.AddSystemNote("workspace add failed: " + err.Error())
			return nil
		}
		if !created {
			a.transcript.AddSystemNote("workspace already exists: " + w.Name)
		}
		a.expanded[w.ID] = true
		a.wsRefresh()
		// Park the tree cursor on the new workspace so ⏎/n act on it.
		for i, r := range a.tree() {
			if r.kind == "ws" && r.workspaceID == w.ID {
				a.treeCursor = i
				break
			}
		}
		if a.debug && created {
			a.writeMeta("ws_add", map[string]any{"id": w.ID, "name": w.Name, "cwd": w.CWD})
		}
		return nil
	}
	var cmd tea.Cmd
	a.wsPromptInput, cmd = a.wsPromptInput.Update(msg)
	return cmd
}

// activateSession loads the session with the given id as the active chat.
// Resets the in-memory transcript, usage, and rotates the log file. If the
// session id matches a brand-new session never seen by claude, firstTurn is
// true so the next send uses --session-id; otherwise --resume picks up.
func (a *App) activateSession(id string) {
	if a.sessStore == nil {
		return
	}
	s, err := a.sessStore.Get(id)
	if err != nil {
		a.transcript.AddSystemNote("session not found: " + err.Error())
		return
	}
	// Closing the previous session's claude turn is harsh; instead, just
	// refuse to switch while a turn is mid-flight to keep state coherent.
	if a.driver != nil || a.spawning {
		a.transcript.AddSystemNote("can't switch session while a turn is running — cancel first (⌃C)")
		return
	}
	_ = a.sessStore.Touch(id)
	a.sessionUUID = s.ID
	a.transcript = Transcript{}
	a.tokens = 0
	a.costUSD = 0
	a.banner = ""
	a.activeCWD = a.sessionCWD(s)
	a.worktree, a.branch = detectWorktreeBranch(a.activeCWD)
	a.openLog()
	// Replay prior turns from the per-session log. driven == true when
	// claude has produced at least one successful result for this session;
	// that's the signal to use --resume on the next spawn. Otherwise we
	// must use --session-id, even on later jflow launches, until claude
	// actually accepts a turn.
	driven := a.hydrateFromLog()
	a.firstTurn = !driven
	a.transcript.AddSystemNote("session " + s.Name + " — " + s.ID[:8])
	// Don't change focus here — the caller decides. Activating from the
	// tree should keep focus in the tree so the user can keep navigating
	// without an extra Tab.
	if a.debug {
		a.writeMeta("session_activate", map[string]any{"id": s.ID, "name": s.Name})
	}
}

// deactivateSession clears the active session — chat hides, composer hides.
func (a *App) deactivateSession() {
	if a.logFile != nil {
		_ = a.logFile.Close()
		a.logFile = nil
	}
	a.sessionUUID = ""
	a.firstTurn = true
	a.transcript = Transcript{}
	a.tokens = 0
	a.costUSD = 0
	a.banner = ""
	a.model = ""
	a.permMode = ""
}

// renderWsPrompt builds the bottom-sheet overlay shown while the workspace
// path prompt is open. Width is the inner-chat-column width (caller indents
// for the column padding). Renders as:
//
//	+ new workspace
//	╭───────────────────────────╮
//	│ /path/to/folder           │
//	╰───────────────────────────╯
//	⏎ create  ·  esc cancel
//
// The rounded box (ComposerBg style) frames the input so it reads as an
// editable field rather than naked indented text. ComposerBg adds 4 cells of
// chrome (border + horizontal padding) — the textinput is sized to the
// remaining inner space.
func (a *App) renderWsPrompt(width int) string {
	if width < 12 {
		width = 12
	}
	const boxChrome = 4
	inputW := width - boxChrome - 1
	if inputW < 8 {
		inputW = 8
	}
	a.wsPromptInput.SetWidth(inputW)
	title := a.theme.Accent.Render("+ new workspace")
	box := a.theme.ComposerBg.Render(a.wsPromptInput.View())
	hint := a.theme.Dim.Render("⏎ create  ·  esc cancel")
	return title + "\n" + box + "\n" + hint
}

// expandHome expands a leading ~ in the given path against $HOME.
func expandHome(p string) string {
	if p == "" || p[0] != '~' {
		return p
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return p
	}
	if p == "~" {
		return home
	}
	if len(p) > 1 && (p[1] == '/' || p[1] == os.PathSeparator) {
		return filepath.Join(home, p[2:])
	}
	return p
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

	case claude.ToolResult:
		text := e.Text
		if text == "" && e.Stdout != "" {
			text = e.Stdout
		}
		if e.IsError && e.Stderr != "" {
			if text != "" {
				text += "\n"
			}
			text += e.Stderr
		}
		a.transcript.AttachToolResult(e.ToolUseID, text, e.IsError)

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

	// Bottom row: a full-width `─` rule (with worktree/branch label) followed
	// by the composer, with the help cheatsheet (when toggled) tucked
	// directly underneath, and a single-line hint bar at the very bottom
	// that surfaces context-aware keybinds for the focused pane.
	rule := a.composerRule(centerW)
	taW := composerInnerW
	if taW < 4 {
		taW = 4
	}
	maxRows := composerMaxRows
	if h3 := a.height / 3; h3 > 0 && h3 < maxRows {
		maxRows = h3
	}
	a.composer.SetMaxHeight(maxRows)
	a.composer.SetWidth(taW)
	composerH := a.composer.Height()
	if composerH < 1 {
		composerH = 1
	}
	taLines := strings.Split(a.composer.View(), "\n")
	composerLines := make([]string, len(taLines))
	for i, l := range taLines {
		prefix := "  "
		if i == 0 {
			prefix = a.theme.Fg.Render("> ")
		}
		composerLines[i] = prefix + l
	}
	bottomBody := indentLines(strings.Join(composerLines, "\n"), chatPadH)
	bottomH := 1 + composerH
	if a.wsPromptOpen {
		overlay := a.renderWsPrompt(innerCenterW)
		oH := strings.Count(overlay, "\n") + 1
		bottomBody += "\n" + indentLines(overlay, chatPadH)
		bottomH += oH
	} else if a.showHelp {
		help := a.helpForFocus()
		sheet := renderHelpSheetWith(a.theme, innerCenterW, help)
		sheetH := strings.Count(sheet, "\n") + 1
		bottomBody += "\n" + indentLines(sheet, chatPadH)
		bottomH += sheetH
	}
	// Mirror the top rule with a plain `─` rule below the composer so the
	// hint bar doesn't sit flush against the input. Keeps the composer feeling
	// like a contained block instead of running straight into the cheatsheet.
	bottomRule := a.theme.Dim.Render(strings.Repeat("─", centerW))
	hintBar := renderHintBar(a, centerW)
	bottom := rule + "\n" + bottomBody + "\n" + bottomRule + "\n" + hintBar
	bottomH += 2

	// Center column: transcript + bottom (composer/sheet). No header, no
	// background fill — the chat column is just terminal default. Reserve
	// one trailing row so the composer doesn't sit on the same baseline as
	// the side panels' bottom border (visually misaligned otherwise).
	const chatBottomPad = 1
	transcriptH := a.height - bottomH - chatBottomPad
	if transcriptH < 3 {
		transcriptH = 3
	}

	// Capture wasAtBottom before resize: when the help sheet opens or the
	// composer grows, transcriptH shrinks. The previous scroll offset is no
	// longer at the bottom of the smaller viewport, so AtBottom() reads false
	// post-resize and the latest messages stay hidden behind the composer/sheet.
	// Reading first lets GotoBottom() fire below.
	wasAtBottom := a.viewport.AtBottom()
	if a.viewport.Height() != transcriptH || a.viewport.Width() != innerCenterW {
		a.viewport.SetWidth(innerCenterW)
		a.viewport.SetHeight(transcriptH)
	}
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
		left := renderLeftPane(a, leftW, a.height)
		right := renderRightPane(a, rightW, a.height)
		content = lipgloss.JoinHorizontal(lipgloss.Top, left, center, right)
	}

	v := tea.NewView(content)
	v.AltScreen = true
	v.MouseMode = tea.MouseModeCellMotion
	return v
}
