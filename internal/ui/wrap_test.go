package ui

import (
	"strings"
	"testing"
)

func TestWrapToWidth(t *testing.T) {
	cases := []struct {
		name  string
		s     string
		width int
		want  []string
	}{
		{
			name: "short line untouched",
			s:    "hello world",
			width: 40,
			want: []string{"hello world"},
		},
		{
			name:  "wraps at word boundary",
			s:     "this is a longer line that needs wrapping",
			width: 20,
			want:  []string{"this is a longer", "line that needs", "wrapping"},
		},
		{
			name:  "preserves hard breaks",
			s:     "first\nsecond line is longer",
			width: 12,
			want:  []string{"first", "second line", "is longer"},
		},
		{
			name:  "hard breaks long word",
			s:     "supercalifragilistic",
			width: 8,
			want:  []string{"supercal", "ifragili", "stic"},
		},
		{
			name:  "Hello message that previously cut off",
			s:     "Hello! Harness test received — looks like everything is wired up correctly. What would you like to work on?",
			width: 60,
			want: []string{
				"Hello! Harness test received — looks like everything is",
				"wired up correctly. What would you like to work on?",
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := strings.Split(wrapToWidth(tc.s, tc.width), "\n")
			if len(got) != len(tc.want) {
				t.Fatalf("len mismatch: got %d lines, want %d. Got=%q", len(got), len(tc.want), got)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Errorf("line %d: got %q, want %q", i, got[i], tc.want[i])
				}
			}
		})
	}
}

func TestWrapWithPrefix(t *testing.T) {
	got := wrapWithPrefix("hello world this is a longer line for prefix wrapping", "you ▸ ", "      ", 30)
	lines := strings.Split(got, "\n")
	if len(lines) < 2 {
		t.Fatalf("expected wrap into >=2 lines, got %d: %q", len(lines), lines)
	}
	if !strings.HasPrefix(lines[0], "you ▸ ") {
		t.Errorf("first line should keep prefix, got %q", lines[0])
	}
	for _, l := range lines[1:] {
		if !strings.HasPrefix(l, "      ") {
			t.Errorf("continuation should start with 6-space indent, got %q", l)
		}
	}
}
