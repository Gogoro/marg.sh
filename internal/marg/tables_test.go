package marg

import (
	"strings"
	"testing"
)

func TestFormatTableBlock_PadsColumns(t *testing.T) {
	in := []string{
		"| Name | Age | City |",
		"|------|-----|------|",
		"| Bob | 30 | NYC |",
		"| Alice | 25 | London |",
	}
	want := []string{
		"| Name  | Age | City   |",
		"| ----- | --- | ------ |",
		"| Bob   | 30  | NYC    |",
		"| Alice | 25  | London |",
	}
	got := formatTableBlock(in)
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("row %d:\n got  %q\n want %q", i, got[i], want[i])
		}
	}
}

func TestFormatTableBlock_HonorsAlignment(t *testing.T) {
	in := []string{
		"| left | center | right |",
		"|:-----|:------:|------:|",
		"| a | b | c |",
	}
	got := formatTableBlock(in)
	if !strings.Contains(got[1], ":") {
		t.Fatalf("alignment colons missing: %q", got[1])
	}
	// Right-aligned column "right" pads on the left.
	if !strings.HasSuffix(strings.TrimSuffix(got[2], " |"), "c") {
		t.Errorf("right-align broken: %q", got[2])
	}
}

func TestFormatTablesInBuffer_SkipsCodeFences(t *testing.T) {
	src := strings.Join([]string{
		"```",
		"| not | a | table |",
		"|-|-|-|",
		"| keep | as | is |",
		"```",
		"",
		"| a | b |",
		"|---|---|",
		"| 1 | hello |",
	}, "\n")
	b := bufferFromString(src)
	formatTablesInBuffer(b)
	got := b.toString()
	if !strings.Contains(got, "| not | a | table |") {
		t.Errorf("table inside fenced block was rewritten:\n%s", got)
	}
	if !strings.Contains(got, "| 1   | hello |") {
		t.Errorf("real table not formatted as expected:\n%s", got)
	}
}

func TestFormatTableBlock_NoTrailingSpaceAfterPipe(t *testing.T) {
	in := []string{
		"| h |",
		"|---|",
		"| x |",
	}
	got := formatTableBlock(in)
	for i, line := range got {
		if strings.HasSuffix(line, " ") {
			t.Errorf("row %d has trailing space: %q", i, line)
		}
	}
}
