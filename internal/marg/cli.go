package marg

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// startTarget describes what marg was asked to open.
type startTarget struct {
	kind targetKind
	path string // absolute path, set for file and dir
}

type targetKind int

const (
	targetDir  targetKind = iota // open the file tree on a directory
	targetFile                   // open the editor on a file
)

func parseArgs(args []string) (startTarget, error) {
	if len(args) == 0 {
		cwd, err := os.Getwd()
		if err != nil {
			return startTarget{}, err
		}
		return startTarget{kind: targetDir, path: cwd}, nil
	}

	if len(args) > 1 {
		return startTarget{}, fmt.Errorf("expected at most one path, got %d", len(args))
	}

	raw := args[0]
	abs, err := filepath.Abs(raw)
	if err != nil {
		return startTarget{}, err
	}

	info, err := os.Stat(abs)
	if err != nil {
		// Path doesn't exist yet — if it looks like a markdown file, treat as new file.
		if os.IsNotExist(err) && isMarkdownPath(abs) {
			return startTarget{kind: targetFile, path: abs}, nil
		}
		return startTarget{}, err
	}

	if info.IsDir() {
		return startTarget{kind: targetDir, path: abs}, nil
	}
	if !isMarkdownPath(abs) {
		return startTarget{}, fmt.Errorf("not a markdown file: %s", raw)
	}
	return startTarget{kind: targetFile, path: abs}, nil
}

func isMarkdownPath(p string) bool {
	ext := strings.ToLower(filepath.Ext(p))
	return ext == ".md" || ext == ".markdown"
}
