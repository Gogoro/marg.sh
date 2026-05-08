package marg

import (
	"errors"
	"os/exec"
	"runtime"
	"strings"
)

// setSystemClipboard copies text to the OS clipboard. On macOS we shell out
// to pbcopy; on Linux we try wl-copy first (Wayland), then xclip, then
// xsel. The minimal-deps approach keeps marg's binary small at the cost of
// requiring one of those tools on Linux.
func setSystemClipboard(text string) error {
	cmd := clipboardWriteCmd()
	if cmd == nil {
		return errors.New("no clipboard tool — install pbcopy / wl-copy / xclip / xsel")
	}
	cmd.Stdin = strings.NewReader(text)
	return cmd.Run()
}

// getSystemClipboard reads text from the OS clipboard.
func getSystemClipboard() (string, error) {
	cmd := clipboardReadCmd()
	if cmd == nil {
		return "", errors.New("no clipboard tool — install pbpaste / wl-paste / xclip / xsel")
	}
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func clipboardWriteCmd() *exec.Cmd {
	if runtime.GOOS == "darwin" {
		if _, err := exec.LookPath("pbcopy"); err == nil {
			return exec.Command("pbcopy")
		}
	}
	if _, err := exec.LookPath("wl-copy"); err == nil {
		return exec.Command("wl-copy")
	}
	if _, err := exec.LookPath("xclip"); err == nil {
		return exec.Command("xclip", "-selection", "clipboard")
	}
	if _, err := exec.LookPath("xsel"); err == nil {
		return exec.Command("xsel", "--clipboard", "--input")
	}
	return nil
}

func clipboardReadCmd() *exec.Cmd {
	if runtime.GOOS == "darwin" {
		if _, err := exec.LookPath("pbpaste"); err == nil {
			return exec.Command("pbpaste")
		}
	}
	if _, err := exec.LookPath("wl-paste"); err == nil {
		return exec.Command("wl-paste", "--no-newline")
	}
	if _, err := exec.LookPath("xclip"); err == nil {
		return exec.Command("xclip", "-selection", "clipboard", "-o")
	}
	if _, err := exec.LookPath("xsel"); err == nil {
		return exec.Command("xsel", "--clipboard", "--output")
	}
	return nil
}
