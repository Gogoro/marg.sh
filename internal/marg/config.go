package marg

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Config holds user-tweakable settings loaded from ~/.config/marg/config.toml.
//
// File format is simple `key = value` per line with `#` comments. Only the
// keys we recognize are used; unknown keys are ignored silently so we can
// add more later without breaking anyone's file.
type Config struct {
	// MaxWidth caps the number of columns used for wrapped text content.
	// 0 means "use the full terminal width". Useful when working in a wide
	// terminal but you want comfortable reading line lengths.
	MaxWidth int

	// SuperRoots are the directories super mode walks when launched without
	// arguments. Defaults to the user's home directory; can be set to one or
	// more absolute paths or `~`-prefixed paths.
	SuperRoots []string

	// IgnoreDirs are extra directory basenames the user wants excluded from
	// every walk on top of the built-in defaults (node_modules, go, Library,
	// etc.). Useful for personal noisy folders.
	IgnoreDirs []string

	// IncludeDirs lists directory basenames that should ALWAYS be searched
	// even when default rules would skip them — typically dot-prefixed dirs
	// like `.claude` or `.obsidian` that contain notes worth opening.
	IncludeDirs []string
}

func defaultConfig() Config {
	cfg := Config{MaxWidth: 0}
	if home, err := os.UserHomeDir(); err == nil {
		cfg.SuperRoots = []string{home}
	}
	return cfg
}

func loadConfig() Config {
	cfg := defaultConfig()
	path, err := configPath()
	if err != nil {
		return cfg
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return cfg
	}
	for _, raw := range strings.Split(string(data), "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		eq := strings.IndexByte(line, '=')
		if eq < 0 {
			continue
		}
		key := strings.TrimSpace(line[:eq])
		value := strings.Trim(strings.TrimSpace(line[eq+1:]), `"'`)
		switch key {
		case "max_width":
			if n, err := strconv.Atoi(value); err == nil && n > 0 {
				cfg.MaxWidth = n
			}
		case "super_roots":
			if roots := parseRootList(value); len(roots) > 0 {
				cfg.SuperRoots = roots
			}
		case "ignore_dirs":
			cfg.IgnoreDirs = parseStringList(value)
		case "include_dirs":
			cfg.IncludeDirs = parseStringList(value)
		}
	}
	return cfg
}

// parseRootList accepts a TOML-style array literal (`["~", "/Users/ole/work"]`)
// and returns each entry, stripped of quotes and with `~` expanded to the
// user's home directory.
func parseRootList(value string) []string {
	home, _ := os.UserHomeDir()
	out := parseStringList(value)
	for i, p := range out {
		if home != "" && (p == "~" || strings.HasPrefix(p, "~/")) {
			out[i] = home + p[1:]
		}
	}
	return out
}

// parseStringList accepts a TOML-style array of strings and returns each
// entry trimmed of quotes and whitespace. Empty entries are dropped.
func parseStringList(value string) []string {
	value = strings.TrimSpace(value)
	value = strings.TrimPrefix(value, "[")
	value = strings.TrimSuffix(value, "]")
	parts := strings.Split(value, ",")
	var out []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		p = strings.Trim(p, `"'`)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func configPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "marg", "config.toml"), nil
}
