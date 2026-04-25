package ui

import (
	"strings"
	"testing"

	"github.com/oobagi/jflow/internal/claude"
)

func TestRenderToolResultFooter_Single(t *testing.T) {
	theme := DefaultTheme()
	out := renderToolResultFooter(theme, "hello-from-probe", false, 80)
	joined := stripANSI(strings.Join(out, "\n"))
	if !strings.Contains(joined, "→ 1 line") {
		t.Errorf("expected header '→ 1 line', got:\n%s", joined)
	}
	if !strings.Contains(joined, "hello-from-probe") {
		t.Errorf("expected body to contain output, got:\n%s", joined)
	}
}

func TestRenderToolResultFooter_TruncatesLongOutput(t *testing.T) {
	theme := DefaultTheme()
	body := strings.Join([]string{"l1", "l2", "l3", "l4", "l5", "l6", "l7", "l8"}, "\n")
	out := renderToolResultFooter(theme, body, false, 80)
	joined := stripANSI(strings.Join(out, "\n"))
	if !strings.Contains(joined, "→ 8 lines") {
		t.Errorf("expected header '→ 8 lines', got:\n%s", joined)
	}
	if !strings.Contains(joined, "(3 more lines)") {
		t.Errorf("expected '(3 more lines)' trailer, got:\n%s", joined)
	}
	if strings.Contains(joined, "l8") {
		t.Errorf("did not expect l8 in preview, got:\n%s", joined)
	}
}

func TestRenderToolResultFooter_Error(t *testing.T) {
	theme := DefaultTheme()
	out := renderToolResultFooter(theme, "boom", true, 80)
	joined := stripANSI(strings.Join(out, "\n"))
	if !strings.Contains(joined, "error · 1 line") {
		t.Errorf("expected error marker, got:\n%s", joined)
	}
}

func TestTranscriptAttachToolResult(t *testing.T) {
	tr := &Transcript{}
	tr.OnContentBlockStart(0, claude.ContentBlock{Type: "tool_use", Name: "Bash", ID: "toolu_1"})
	tr.OnContentBlockStop(0)
	tr.AttachToolResult("toolu_1", "hello", false)

	if len(tr.Blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(tr.Blocks))
	}
	b := tr.Blocks[0]
	if !b.HasToolResult || b.ToolResult != "hello" {
		t.Errorf("expected tool result attached, got %+v", b)
	}

	// A nonsense ID should be a no-op (and not panic).
	tr.AttachToolResult("toolu_does_not_exist", "x", false)

	rendered := tr.Render(DefaultTheme(), 80)
	if !strings.Contains(stripANSI(rendered), "→ 1 line") {
		t.Errorf("expected rendered transcript to include footer, got:\n%s", rendered)
	}
}

// stripANSI removes lipgloss escape codes so tests can match plain text.
func stripANSI(s string) string {
	var out strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == 0x1b {
			// skip CSI ... letter
			j := i + 1
			for j < len(s) {
				c := s[j]
				j++
				if (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') {
					break
				}
			}
			i = j - 1
			continue
		}
		out.WriteByte(s[i])
	}
	return out.String()
}
