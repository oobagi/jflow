package claude

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
)

// SpawnOpts configures a single `claude -p` invocation.
// Each invocation handles one user→assistant turn.
type SpawnOpts struct {
	SessionID          string // required uuid; --session-id on first call, --resume thereafter
	Resume             bool   // false → use --session-id; true → use --resume
	Prompt             string // first/next user message; passed as positional arg
	Model              string // optional, e.g. "sonnet" / "opus"
	Bare               bool   // --bare for tool-program sessions
	SystemPromptFile   string // --system-prompt-file
	AppendSystemPrompt string // --append-system-prompt
	AllowedTools       []string
	Tools              []string
	PermissionMode     string // default | acceptEdits | auto | plan | dontAsk | bypassPermissions
	MaxTurns           int
	MaxBudgetUSD       float64
	AddDirs            []string
	CWD                string
	Effort             string // low | medium | high | xhigh | max

	// LogWriter, if non-nil, receives every raw stdout JSONL line
	// (each line followed by a single '\n'). Useful for "what happened
	// in my last session" replay/debug. Errors writing are ignored.
	LogWriter io.Writer
}

// Driver wraps a running `claude -p` subprocess. Events stream out of Events()
// until the subprocess exits, at which point the channel is closed and a
// DriverExit{} event is emitted.
type Driver struct {
	SessionID string
	cmd       *exec.Cmd
	events    chan Event
	stdinSink io.Closer // /dev/null handle (v0); becomes a real pipe in v0.5
	logWriter io.Writer
	argv      []string

	// stderr captures the subprocess's stderr stream so it doesn't leak into
	// the TUI's alt-screen. The tail is surfaced via DriverExit when the
	// process exits with an error.
	stderrMu  sync.Mutex
	stderrBuf bytes.Buffer
}

// Argv returns the full argument list passed to the claude subprocess
// (including "claude" at index 0). Useful for debug logging.
func (d *Driver) Argv() []string { return d.argv }

func Spawn(ctx context.Context, opts SpawnOpts) (*Driver, error) {
	args := []string{
		"-p",
		"--output-format", "stream-json",
		"--verbose",
		"--include-partial-messages",
	}
	if opts.Resume {
		args = append(args, "--resume", opts.SessionID)
	} else {
		args = append(args, "--session-id", opts.SessionID)
	}
	if opts.Bare {
		args = append(args, "--bare")
	}
	if opts.Model != "" {
		args = append(args, "--model", opts.Model)
	}
	if opts.Effort != "" {
		args = append(args, "--effort", opts.Effort)
	}
	if opts.PermissionMode != "" {
		args = append(args, "--permission-mode", opts.PermissionMode)
	}
	if opts.SystemPromptFile != "" {
		args = append(args, "--system-prompt-file", opts.SystemPromptFile)
	}
	if opts.AppendSystemPrompt != "" {
		args = append(args, "--append-system-prompt", opts.AppendSystemPrompt)
	}
	if len(opts.AllowedTools) > 0 {
		args = append(args, "--allowed-tools")
		args = append(args, opts.AllowedTools...)
	}
	if len(opts.Tools) > 0 {
		args = append(args, "--tools")
		args = append(args, opts.Tools...)
	}
	if opts.MaxTurns > 0 {
		args = append(args, "--max-turns", strconv.Itoa(opts.MaxTurns))
	}
	if opts.MaxBudgetUSD > 0 {
		args = append(args, "--max-budget-usd", strconv.FormatFloat(opts.MaxBudgetUSD, 'f', 2, 64))
	}
	for _, d := range opts.AddDirs {
		args = append(args, "--add-dir", d)
	}
	if opts.Prompt != "" {
		args = append(args, opts.Prompt)
	}

	cmd := exec.CommandContext(ctx, "claude", args...)
	if opts.CWD != "" {
		cmd.Dir = opts.CWD
	}
	// Point stdin at /dev/null so claude immediately sees EOF instead of
	// waiting 3s for input. v0.5 will swap this for a real pipe when we
	// enable --input-format=stream-json for mid-flight user injection.
	devNull, err := os.Open(os.DevNull)
	if err != nil {
		return nil, fmt.Errorf("open /dev/null: %w", err)
	}
	cmd.Stdin = devNull
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("stdout pipe: %w", err)
	}
	d := &Driver{
		SessionID: opts.SessionID,
		cmd:       cmd,
		events:    make(chan Event, 64),
		stdinSink: devNull,
		logWriter: opts.LogWriter,
		argv:      append([]string{"claude"}, args...),
	}
	cmd.Stderr = &lockedWriter{mu: &d.stderrMu, w: &d.stderrBuf}
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start claude: %w", err)
	}
	go d.read(stdout)
	return d, nil
}

// lockedWriter serializes writes from the subprocess's stderr goroutine
// with reads done from the main TUI goroutine.
type lockedWriter struct {
	mu *sync.Mutex
	w  io.Writer
}

func (l *lockedWriter) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.w.Write(p)
}

func (d *Driver) Events() <-chan Event { return d.events }

func (d *Driver) read(r io.Reader) {
	defer close(d.events)
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 64*1024), 8*1024*1024) // tolerate big lines
	for sc.Scan() {
		line := sc.Bytes()
		if len(line) == 0 {
			continue
		}
		if d.logWriter != nil {
			_, _ = d.logWriter.Write(line)
			_, _ = d.logWriter.Write([]byte{'\n'})
		}
		ev, err := parseLine(line)
		if err != nil {
			d.events <- ParseError{Err: err, Line: append([]byte(nil), line...)}
			continue
		}
		if ev != nil {
			d.events <- ev
		}
	}
	if err := sc.Err(); err != nil {
		d.events <- ParseError{Err: err}
	}
	exitErr := d.cmd.Wait()
	var stderrTail string
	if exitErr != nil {
		stderrTail = d.tailStderr()
	}
	d.events <- DriverExit{Err: exitErr, Stderr: stderrTail}
}

// tailStderr returns up to the last ~1KB of captured stderr, trimmed.
func (d *Driver) tailStderr() string {
	d.stderrMu.Lock()
	defer d.stderrMu.Unlock()
	b := d.stderrBuf.Bytes()
	const max = 1024
	if len(b) > max {
		b = b[len(b)-max:]
	}
	return strings.TrimSpace(string(b))
}

// decodeToolResultContent extracts text from a tool_result `content`
// field. Claude sends either a bare string (Bash/Read/etc.) or an array
// of {type:text|image, ...} parts; image parts are skipped.
func decodeToolResultContent(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s
	}
	var parts []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	if err := json.Unmarshal(raw, &parts); err == nil {
		var out strings.Builder
		for _, p := range parts {
			if p.Type == "text" && p.Text != "" {
				if out.Len() > 0 {
					out.WriteByte('\n')
				}
				out.WriteString(p.Text)
			}
		}
		return out.String()
	}
	return ""
}

func parseLine(line []byte) (Event, error) {
	var env Envelope
	if err := json.Unmarshal(line, &env); err != nil {
		return nil, fmt.Errorf("unmarshal envelope: %w", err)
	}
	switch env.Type {
	case "system":
		switch env.Subtype {
		case "init":
			return SystemInit{
				CWD:               env.CWD,
				Model:             env.Model,
				PermissionMode:    env.PermissionMode,
				Tools:             env.Tools,
				MCPServers:        env.MCPServers,
				SlashCommands:     env.SlashCommands,
				ClaudeCodeVersion: env.ClaudeCodeVersion,
				SessionID:         env.SessionID,
			}, nil
		case "status":
			return SystemStatus{Status: env.Status}, nil
		case "hook_started":
			return HookStarted{HookName: env.HookName, HookEvent: env.HookEvent, HookID: env.HookID}, nil
		case "hook_response":
			return HookResponse{
				HookName: env.HookName, HookEvent: env.HookEvent, HookID: env.HookID,
				Outcome: env.Outcome, ExitCode: env.ExitCode,
				Stdout: env.Stdout, Stderr: env.Stderr,
			}, nil
		}
	case "stream_event":
		var sev StreamEvent
		if len(env.Event) == 0 {
			return nil, nil
		}
		if err := json.Unmarshal(env.Event, &sev); err != nil {
			return nil, fmt.Errorf("stream_event payload: %w", err)
		}
		switch sev.Type {
		case "message_start":
			var (
				u     Usage
				mid   string
				model string
			)
			if sev.Message != nil {
				mid = sev.Message.ID
				model = sev.Message.Model
				if sev.Message.Usage != nil {
					u = *sev.Message.Usage
				}
			}
			return MessageStart{MessageID: mid, Model: model, Usage: u, TTFTMs: env.TTFTMs}, nil
		case "content_block_start":
			if sev.ContentBlock == nil {
				return nil, nil
			}
			return ContentBlockStart{Index: sev.Index, Block: *sev.ContentBlock}, nil
		case "content_block_delta":
			if sev.Delta == nil {
				return nil, nil
			}
			return ContentBlockDelta{Index: sev.Index, Delta: *sev.Delta}, nil
		case "content_block_stop":
			return ContentBlockStop{Index: sev.Index}, nil
		case "message_delta":
			var (
				stop string
				u    Usage
			)
			if sev.Delta != nil {
				stop = sev.Delta.StopReason
			}
			if sev.Usage != nil {
				u = *sev.Usage
			}
			return MessageDelta{StopReason: stop, Usage: u}, nil
		case "message_stop":
			return MessageStop{}, nil
		}
	case "assistant":
		var msg AssistantMsg
		if len(env.Message) > 0 {
			if err := json.Unmarshal(env.Message, &msg); err != nil {
				return nil, fmt.Errorf("assistant payload: %w", err)
			}
		}
		return AssistantSnapshot{Message: msg}, nil
	case "user":
		// `user` events carry either an echo of the prompt (with
		// --replay-user-messages) or a tool_result block fed back to the
		// model after claude executed a tool. Distinguish by inspecting
		// the message content array.
		if len(env.Message) > 0 {
			var um UserMsg
			if err := json.Unmarshal(env.Message, &um); err == nil {
				for _, cb := range um.Content {
					if cb.Type != "tool_result" {
						continue
					}
					tr := ToolResult{
						ToolUseID: cb.ToolUseID,
						IsError:   cb.IsError,
						Text:      decodeToolResultContent(cb.Content),
					}
					if env.ToolUseResult != nil {
						tr.Stdout = env.ToolUseResult.Stdout
						tr.Stderr = env.ToolUseResult.Stderr
					}
					return tr, nil
				}
			}
		}
		return UserEcho{}, nil
	case "rate_limit_event":
		if env.RateLimitInfo == nil {
			return nil, nil
		}
		return RateLimit{Info: *env.RateLimitInfo}, nil
	case "result":
		return Result{
			Subtype:        env.Subtype,
			IsError:        env.IsError,
			DurationMS:     env.DurationMS,
			NumTurns:       env.NumTurns,
			Result:         env.Result,
			StopReason:     env.StopReason,
			TotalCostUSD:   env.TotalCostUSD,
			ModelUsage:     env.ModelUsage,
			TerminalReason: env.TerminalReason,
		}, nil
	}
	return nil, nil
}

// Interrupt sends SIGINT to the running claude subprocess. The process should
// emit a partial result and exit.
func (d *Driver) Interrupt() error {
	if d.cmd != nil && d.cmd.Process != nil {
		return d.cmd.Process.Signal(os.Interrupt)
	}
	return nil
}

// Close releases driver resources. The subprocess exits on its own when claude
// finishes its turn; this just tidies up the stdin handle.
func (d *Driver) Close() error {
	if d.stdinSink != nil {
		_ = d.stdinSink.Close()
	}
	return nil
}
