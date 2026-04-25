package ui

import (
	"fmt"
	"strings"

	"github.com/oobagi/jflow/internal/claude"
)

// BlockKind identifies the rendered category of a transcript block.
type BlockKind int

const (
	BlockUser BlockKind = iota
	BlockText           // assistant text
	BlockThinking
	BlockToolUse
	BlockSystem // info/banner (init, compaction, errors)
	BlockTiming // dim "worked for 2.3s" line under the previous block
)

// Block is one rendered unit in the transcript.
type Block struct {
	Kind      BlockKind
	Text      string // for User/Text/Thinking/System/Timing
	ToolName  string // for ToolUse
	ToolID    string
	ToolArgs  string // streamed partial JSON, prettified at stop
	Sealed    bool   // false while streaming, true after stop
	StreamIdx int    // matches stream_event index for content blocks
}

// Transcript holds the ordered list of blocks for the current session.
type Transcript struct {
	Blocks []*Block
}

// AddUserMessage appends a sealed user block.
func (t *Transcript) AddUserMessage(text string) {
	t.Blocks = append(t.Blocks, &Block{Kind: BlockUser, Text: text, Sealed: true})
}

// AddSystemNote appends a sealed system note (e.g. "session started", error).
func (t *Transcript) AddSystemNote(text string) {
	t.Blocks = append(t.Blocks, &Block{Kind: BlockSystem, Text: text, Sealed: true})
}

// AddTiming appends a "worked for 2.3s" line that renders dimly under the
// previous block with no blank separator above it.
func (t *Transcript) AddTiming(text string) {
	t.Blocks = append(t.Blocks, &Block{Kind: BlockTiming, Text: text, Sealed: true})
}

// OnContentBlockStart starts a new streaming block keyed by stream index.
func (t *Transcript) OnContentBlockStart(idx int, cb claude.ContentBlock) {
	b := &Block{StreamIdx: idx}
	switch cb.Type {
	case "text":
		b.Kind = BlockText
		b.Text = cb.Text
	case "thinking":
		b.Kind = BlockThinking
		b.Text = cb.Text
	case "tool_use":
		b.Kind = BlockToolUse
		b.ToolName = cb.Name
		b.ToolID = cb.ID
		if len(cb.Input) > 0 {
			b.ToolArgs = string(cb.Input)
		}
	default:
		b.Kind = BlockSystem
		b.Text = "(unknown content block: " + cb.Type + ")"
	}
	t.Blocks = append(t.Blocks, b)
}

// OnContentBlockDelta appends streamed content to the matching block.
func (t *Transcript) OnContentBlockDelta(idx int, d claude.ContentDelta) {
	b := t.findStreamingBlock(idx)
	if b == nil {
		return
	}
	switch d.Type {
	case "text_delta":
		b.Text += d.Text
	case "thinking_delta":
		b.Text += d.Thinking
	case "input_json_delta":
		b.ToolArgs += d.PartialJSON
	}
}

// OnContentBlockStop seals the matching block.
func (t *Transcript) OnContentBlockStop(idx int) {
	if b := t.findStreamingBlock(idx); b != nil {
		b.Sealed = true
	}
}

func (t *Transcript) findStreamingBlock(idx int) *Block {
	for i := len(t.Blocks) - 1; i >= 0; i-- {
		b := t.Blocks[i]
		if !b.Sealed && b.StreamIdx == idx {
			return b
		}
	}
	return nil
}

// Render produces the full transcript as a string, wrapped to width.
// Naive: no scrolling viewport yet. v0 displays the latest N lines.
//
// Prefix-aware wrapping: continuation lines align under the first line so
// "claude ▸ <long text…>" wraps cleanly with a 9-column hanging indent.
func (t *Transcript) Render(theme Theme, width int) string {
	if width < 20 {
		width = 20
	}
	var lines []string
	for i, b := range t.Blocks {
		switch b.Kind {
		case BlockUser:
			rendered := wrapWithPrefix(b.Text, theme.UserPrefix.Render("you ▸ "), "      ", width)
			lines = append(lines, strings.Split(rendered, "\n")...)
		case BlockText:
			rendered := wrapWithPrefix(b.Text, theme.AsstPrefix.Render("claude ▸ "), "         ", width)
			lines = append(lines, strings.Split(rendered, "\n")...)
		case BlockThinking:
			lines = append(lines, theme.Thinking.Render("✱ thinking"))
			body := wrapToWidth(b.Text, width-2)
			for _, p := range strings.Split(body, "\n") {
				lines = append(lines, theme.Thinking.Render("│ "+p))
			}
		case BlockToolUse:
			lines = append(lines, theme.ToolHeader.Render(fmt.Sprintf("⚙ %s", b.ToolName)))
			args := strings.TrimSpace(b.ToolArgs)
			if args == "" {
				args = "{}"
			}
			body := wrapToWidth(args, width-2)
			for _, p := range strings.Split(body, "\n") {
				lines = append(lines, theme.ToolBody.Render("│ "+p))
			}
		case BlockSystem:
			body := wrapWithPrefix(b.Text, "· ", "  ", width)
			for _, p := range strings.Split(body, "\n") {
				lines = append(lines, theme.Dim.Render(p))
			}
		case BlockTiming:
			// Indent matches the assistant body indent so it tucks neatly
			// under "claude ▸ ...".
			lines = append(lines, theme.Dim.Render("         worked for "+b.Text))
		}
		// Suppress the trailing blank when the next block is a timing line
		// so the timing sits directly under its response.
		nextIsTiming := i+1 < len(t.Blocks) && t.Blocks[i+1].Kind == BlockTiming
		if !nextIsTiming {
			lines = append(lines, "")
		}
	}
	return strings.Join(lines, "\n")
}
