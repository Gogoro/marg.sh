package marg

import (
	"testing"
)

func TestTSLanguageFor(t *testing.T) {
	for _, name := range []string{
		"sql", "postgres", "go", "golang",
		"js", "javascript", "jsx", "rust", "rs",
		"typescript", "ts", "python", "py",
		"bash", "sh", "shell", "yaml", "yml",
	} {
		if tsLanguageFor(name) == nil {
			t.Errorf("expected language for hint %q, got nil", name)
		}
	}
	if tsLanguageFor("haskell") != nil {
		t.Error("did not expect a language for haskell")
	}
}

func TestTSTokenizeBlock_SQL(t *testing.T) {
	spans := map[int][]tokenSpan{}
	src := "SELECT id, name FROM users WHERE id = 1"
	if !tsTokenizeBlock(spans, tsLanguageFor("sql"), src, 0) {
		t.Fatal("sql highlighter did not run")
	}
	if len(spans[0]) == 0 {
		t.Fatal("expected at least one span on row 0")
	}
}

func TestTSTokenizeBlock_Go(t *testing.T) {
	spans := map[int][]tokenSpan{}
	src := "package main\nfunc main() { return }\n"
	if !tsTokenizeBlock(spans, tsLanguageFor("go"), src, 0) {
		t.Fatal("go highlighter did not run")
	}
	totalSpans := 0
	for _, list := range spans {
		totalSpans += len(list)
	}
	if totalSpans == 0 {
		t.Fatal("expected captures from go highlighter")
	}
}

func TestTSTokenizeBlock_Rust(t *testing.T) {
	spans := map[int][]tokenSpan{}
	src := "fn main() { let x = 42; println!(\"{}\", x); }"
	if !tsTokenizeBlock(spans, tsLanguageFor("rust"), src, 0) {
		t.Fatal("rust highlighter did not run")
	}
	if len(spans[0]) == 0 {
		t.Fatal("expected captures from rust highlighter")
	}
}

func TestTSTokenizeBlock_JS(t *testing.T) {
	spans := map[int][]tokenSpan{}
	src := "function greet(name) { return `hello, ${name}`; }"
	if !tsTokenizeBlock(spans, tsLanguageFor("javascript"), src, 0) {
		t.Fatal("js highlighter did not run")
	}
	if len(spans[0]) == 0 {
		t.Fatal("expected captures from js highlighter")
	}
}

func TestTSTokenizeBlock_BufferRowOffset(t *testing.T) {
	spans := map[int][]tokenSpan{}
	src := "SELECT 1"
	tsTokenizeBlock(spans, tsLanguageFor("sql"), src, 17)
	if len(spans[17]) == 0 {
		t.Fatal("expected span on row 17 (the buffer-row offset)")
	}
	if len(spans[0]) != 0 {
		t.Fatal("did not expect span on row 0 when offset was 17")
	}
}

func TestTSTokenizeBlock_MultilineString(t *testing.T) {
	// Go raw string spanning 3 rows — verify spans land on each row.
	spans := map[int][]tokenSpan{}
	src := "package x\nvar s = `line1\nline2\nline3`"
	if !tsTokenizeBlock(spans, tsLanguageFor("go"), src, 0) {
		t.Fatal("go highlighter did not run")
	}
	rowsWithSpans := 0
	for row := 1; row <= 3; row++ {
		if len(spans[row]) > 0 {
			rowsWithSpans++
		}
	}
	if rowsWithSpans < 2 {
		t.Fatalf("expected raw-string token to land on multiple rows; rows hit: %d", rowsWithSpans)
	}
}

func TestTSTokenizeBlock_Python(t *testing.T) {
	spans := map[int][]tokenSpan{}
	src := "def greet(name: str) -> str:\n    return f\"hello, {name}\"\n"
	if !tsTokenizeBlock(spans, tsLanguageFor("python"), src, 0) {
		t.Fatal("python highlighter did not run")
	}
	if len(spans[0]) == 0 {
		t.Fatal("expected captures on row 0 from python highlighter")
	}
}

func TestTSTokenizeBlock_TypeScript(t *testing.T) {
	spans := map[int][]tokenSpan{}
	src := "interface User { name: string; age: number; }\nconst u: User = { name: \"a\", age: 1 };"
	if !tsTokenizeBlock(spans, tsLanguageFor("typescript"), src, 0) {
		t.Fatal("typescript highlighter did not run")
	}
	if len(spans[0]) == 0 {
		t.Fatal("expected captures from typescript highlighter")
	}
}

func TestTSTokenizeBlock_Bash(t *testing.T) {
	spans := map[int][]tokenSpan{}
	src := "#!/bin/bash\nfor f in *.md; do\n  echo \"$f\"\ndone\n"
	if !tsTokenizeBlock(spans, tsLanguageFor("bash"), src, 0) {
		t.Fatal("bash highlighter did not run")
	}
	totalSpans := 0
	for _, list := range spans {
		totalSpans += len(list)
	}
	if totalSpans == 0 {
		t.Fatal("expected captures from bash highlighter")
	}
}

func TestTSTokenizeBlock_YAML(t *testing.T) {
	spans := map[int][]tokenSpan{}
	src := "name: marg\nversion: 0.1.0\nflag: true\n"
	if !tsTokenizeBlock(spans, tsLanguageFor("yaml"), src, 0) {
		t.Fatal("yaml highlighter did not run")
	}
	totalSpans := 0
	for _, list := range spans {
		totalSpans += len(list)
	}
	if totalSpans == 0 {
		t.Fatal("expected captures from yaml highlighter")
	}
}

func TestScanCodeBlocksIntegration(t *testing.T) {
	doc := "intro paragraph\n\n```sql\nSELECT * FROM users;\n```\n\nmore prose\n\n```go\npackage x\nfunc f() {}\n```\n"
	e := editor{buf: bufferFromString(doc), mode: modeNormal}
	out := e.scanCodeBlocks()

	// SQL block: opening fence at row 2, content at row 3, closing fence at row 4.
	if len(out.spans[3]) == 0 {
		t.Fatal("expected SQL block to produce token spans on row 3")
	}
	// Go block: fence at row 8, content at rows 9-10, closing fence at row 11.
	if len(out.spans[9]) == 0 {
		t.Fatal("expected Go block to produce token spans on row 9")
	}
}
