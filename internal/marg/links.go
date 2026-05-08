package marg

import (
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// followLinkAtCursor implements `gd` in normal mode: jump to the markdown
// link the cursor is currently sitting on. External URLs (with a scheme
// like http://, https://, mailto:) open in the system browser; everything
// else is treated as a path resolved relative to the open file's
// directory and opened as a buffer.
func (e *editor) followLinkAtCursor() tea.Cmd {
	line := e.buf.line(e.row)
	span, ok := linkSpanAtCol(line, e.col)
	if !ok {
		e.flash = "no link under cursor"
		return nil
	}
	url := strings.TrimSpace(string(line[span.textEnd+2 : span.end-1]))
	if url == "" {
		e.flash = "empty link target"
		return nil
	}
	if hasURLScheme(url) {
		if err := openInBrowser(url); err != nil {
			e.flash = "open failed: " + err.Error()
			return nil
		}
		e.flash = "opened in browser"
		return nil
	}
	target := resolveInternalLink(e.filepath, url)
	if target == "" {
		e.flash = "can't resolve link target"
		return nil
	}
	return openFileCmd(target)
}

// linkSpanAtCol returns the link span that contains the given column
// (rune index) on the line, if any.
func linkSpanAtCol(line []rune, col int) (linkSpan, bool) {
	for _, s := range linkSpans(line) {
		if col >= s.start && col < s.end {
			return s, true
		}
	}
	return linkSpan{}, false
}

// hasURLScheme reports whether s looks like a URL with an explicit scheme
// (e.g. `https://example.com`, `mailto:foo@bar`). The check follows RFC
// 3986: a scheme starts with a letter and is followed by letters, digits,
// `+`, `-`, or `.`, ending in a colon.
func hasURLScheme(s string) bool {
	colon := strings.IndexByte(s, ':')
	if colon <= 0 {
		return false
	}
	if !isASCIILetter(s[0]) {
		return false
	}
	for i := 1; i < colon; i++ {
		c := s[i]
		if !isASCIILetter(c) && !isASCIIDigit(c) && c != '+' && c != '-' && c != '.' {
			return false
		}
	}
	return true
}

func isASCIILetter(b byte) bool { return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') }
func isASCIIDigit(b byte) bool  { return b >= '0' && b <= '9' }

// resolveInternalLink turns a relative-to-doc link target into an
// absolute path. Strips any `#anchor` fragment, resolves relative paths
// against the current file's directory, and adds `.md` when the target
// has no extension. Returns "" if there's no current file context to
// resolve against.
func resolveInternalLink(currentFile, target string) string {
	if i := strings.IndexByte(target, '#'); i >= 0 {
		target = target[:i]
	}
	if target == "" {
		return ""
	}
	if !filepath.IsAbs(target) {
		if currentFile == "" {
			return ""
		}
		target = filepath.Join(filepath.Dir(currentFile), target)
	}
	if filepath.Ext(target) == "" {
		target += ".md"
	}
	return target
}

func openInBrowser(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", "", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	return cmd.Start()
}
