package marg

import (
	"os/exec"
	"strings"
)

// findMarkdownFiles enumerates every `.md` / `.markdown` file under any of
// the given roots. It tries the fastest tool available and falls back to a
// pure-Go walk if none are installed.
//
// Order:
//  1. `fd` — fastest in practice, multi-threaded, respects .gitignore
//  2. `rg --files` — also fast, also respects .gitignore
//  3. built-in walker — slow but always available
func findMarkdownFiles(roots []string) []string {
	if len(roots) == 0 {
		return nil
	}
	if path, err := exec.LookPath("fd"); err == nil {
		if out, err := walkWithFd(path, roots); err == nil {
			return out
		}
	}
	if path, err := exec.LookPath("rg"); err == nil {
		if out, err := walkWithRg(path, roots); err == nil {
			return out
		}
	}
	var all []string
	for _, r := range roots {
		all = append(all, collectMarkdownFiles(r)...)
	}
	return all
}

func walkWithFd(bin string, roots []string) ([]string, error) {
	args := []string{
		"--type", "f",
		"--extension", "md",
		"--extension", "markdown",
		"--exclude", "node_modules",
		"--exclude", "vendor",
		"--exclude", "Library",
		"--exclude", "Applications",
		"--exclude", "target",
		"--exclude", "build",
		"--exclude", "dist",
		".",
	}
	args = append(args, roots...)
	out, err := exec.Command(bin, args...).Output()
	if err != nil {
		return nil, err
	}
	return splitLines(out), nil
}

func walkWithRg(bin string, roots []string) ([]string, error) {
	args := []string{
		"--files",
		"--type", "md",
		"--glob", "!node_modules",
		"--glob", "!vendor",
		"--glob", "!Library",
		"--glob", "!Applications",
		"--glob", "!target",
		"--glob", "!build",
		"--glob", "!dist",
	}
	args = append(args, roots...)
	out, err := exec.Command(bin, args...).Output()
	if err != nil {
		return nil, err
	}
	return splitLines(out), nil
}

func splitLines(b []byte) []string {
	s := strings.TrimRight(string(b), "\n")
	if s == "" {
		return nil
	}
	return strings.Split(s, "\n")
}
