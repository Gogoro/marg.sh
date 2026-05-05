package marg

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func keyRunes(s string) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
}

func keySpecial(t tea.KeyType) tea.KeyMsg {
	return tea.KeyMsg{Type: t}
}

func TestEditorTypingInInsertMode(t *testing.T) {
	e := newEditor("")
	e.resize(80, 24)
	// enter insert mode
	e, _ = e.update(keyRunes("i"))
	if e.mode != modeInsert {
		t.Fatalf("expected insert mode, got %v", e.mode)
	}
	for _, r := range "hello" {
		e, _ = e.update(keyRunes(string(r)))
	}
	if got := e.buf.toString(); got != "hello" {
		t.Fatalf("want %q, got %q", "hello", got)
	}
	if !e.dirty {
		t.Fatal("expected dirty after typing")
	}
}

func TestEditorEscapeReturnsToNormal(t *testing.T) {
	e := newEditor("")
	e.resize(80, 24)
	e, _ = e.update(keyRunes("i"))
	e, _ = e.update(keyRunes("a"))
	e, _ = e.update(keySpecial(tea.KeyEsc))
	if e.mode != modeNormal {
		t.Fatalf("expected normal, got %v", e.mode)
	}
}

func TestUndoRedoRoundtrip(t *testing.T) {
	e := newEditor("")
	e.resize(80, 24)

	e, _ = e.update(keyRunes("i"))
	for _, r := range "hello" {
		e, _ = e.update(keyRunes(string(r)))
	}
	e, _ = e.update(keySpecial(tea.KeyEsc))
	if got := e.buf.toString(); got != "hello" {
		t.Fatalf("after typing want %q, got %q", "hello", got)
	}

	e, _ = e.update(keyRunes("u"))
	if got := e.buf.toString(); got != "" {
		t.Fatalf("after undo want empty, got %q", got)
	}

	e, _ = e.update(tea.KeyMsg{Type: tea.KeyCtrlR})
	if got := e.buf.toString(); got != "hello" {
		t.Fatalf("after redo want %q, got %q", "hello", got)
	}
}

func TestUndoAfterDD(t *testing.T) {
	e := newEditor("")
	e.resize(80, 24)
	e.buf = bufferFromString("first\nsecond\nthird")
	e.initHistory()
	e.history = nil
	e.initHistory()

	// Move to second line, dd it
	e, _ = e.update(keyRunes("j"))
	e, _ = e.update(keyRunes("d"))
	e, _ = e.update(keyRunes("d"))
	if got := e.buf.toString(); got != "first\nthird" {
		t.Fatalf("after dd want %q, got %q", "first\nthird", got)
	}

	e, _ = e.update(keyRunes("u"))
	if got := e.buf.toString(); got != "first\nsecond\nthird" {
		t.Fatalf("after undo want all 3 lines, got %q", got)
	}
}

func TestSoftWrapHardBreaksLongWord(t *testing.T) {
	e := newEditor("")
	e.resize(20, 10)
	longWord := "Pneumonoultramicroscopicsilicovolcanoconiosis" // 45 chars, no spaces
	e.buf = bufferFromString(longWord)
	visuals := e.allVisualLines()
	if len(visuals) < 2 {
		t.Fatalf("expected hard wrap, got %d visual lines", len(visuals))
	}
	var combined []rune
	for _, v := range visuals {
		combined = append(combined, e.buf.line(v.row)[v.startCol:v.endCol]...)
	}
	if string(combined) != longWord {
		t.Fatalf("hard wrap dropped runes: %q", string(combined))
	}
}

func TestSoftWrapSplitsLongLine(t *testing.T) {
	e := newEditor("")
	e.resize(20, 10)
	e.buf = bufferFromString("the quick brown fox jumps over the lazy dog and keeps running for a while")
	visuals := e.allVisualLines()
	if len(visuals) < 2 {
		t.Fatalf("expected wrap, got %d visual lines", len(visuals))
	}
	// All visuals must belong to row 0.
	for _, v := range visuals {
		if v.row != 0 {
			t.Fatalf("unexpected row %d", v.row)
		}
	}
}
