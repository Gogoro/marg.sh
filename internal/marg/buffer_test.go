package marg

import "testing"

func TestBufferInsertAndDelete(t *testing.T) {
	b := newBuffer()
	b.insertRune(0, 0, 'h')
	b.insertRune(0, 1, 'i')
	if got := b.toString(); got != "hi" {
		t.Fatalf("want %q, got %q", "hi", got)
	}
	b.insertNewline(0, 1)
	if got := b.toString(); got != "h\ni" {
		t.Fatalf("want %q, got %q", "h\ni", got)
	}
	row, col := b.deleteRuneBefore(1, 0)
	if row != 0 || col != 1 {
		t.Fatalf("join: want (0,1), got (%d,%d)", row, col)
	}
	if got := b.toString(); got != "hi" {
		t.Fatalf("after join want %q, got %q", "hi", got)
	}
}

func TestBufferUnicode(t *testing.T) {
	b := bufferFromString("héllo")
	if b.lineLen(0) != 5 {
		t.Fatalf("want 5 runes, got %d", b.lineLen(0))
	}
	b.insertRune(0, 1, 'ë')
	if got := b.toString(); got != "hëéllo" {
		t.Fatalf("got %q", got)
	}
}

func TestBufferWordCount(t *testing.T) {
	b := bufferFromString("hello world\n  foo  bar baz\n")
	if got := b.wordCount(); got != 5 {
		t.Fatalf("want 5, got %d", got)
	}
}
