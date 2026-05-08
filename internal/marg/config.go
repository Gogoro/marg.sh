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

	// CodeMaxWidth caps the wrap width specifically for fenced code blocks
	// and table rows. 0 means "let them use the full terminal width" — the
	// usual choice, so prose stays narrow at MaxWidth while code and tables
	// claim whatever horizontal room they need.
	CodeMaxWidth int

	// CenterAbove is the terminal width at which the text block starts
	// being horizontally centered. 0 disables centering entirely (text
	// hugs the left edge). A typical value: 160.
	CenterAbove int

	// CodeTheme picks the Chroma style used for syntax-highlighted fenced
	// code blocks. Any name from `chroma --list` works. Defaults to
	// "monokai" (high-contrast, readable on dark terminals).
	CodeTheme string

	// Theme switches the entire editor palette. "dark" (default), "light",
	// or "sepia". Light and sepia paint an opaque background so they look
	// right even when launched from a dark terminal window.
	Theme string

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

	// AI holds settings for the AI-assisted features (proofread, future
	// rewrite/summarize). Populated from the `[ai]` section of the config.
	AI AIConfig
}

// AIConfig is the model + auth settings for AI features. Two role-based
// model slots — fast for inline mechanical work (proofread), smart for
// substantive passes (future :proof %, paragraph rewrite). Each feature
// picks one slot, so adding new features doesn't bloat the config.
type AIConfig struct {
	// APIKey for api.anthropic.com. Empty falls back to the
	// ANTHROPIC_API_KEY environment variable. If neither is set, AI
	// features are off and `:proof` flashes a hint.
	APIKey string

	// FastModel is used for inline, latency-sensitive work — current
	// `:proof` runs against this slot. Defaults to claude-haiku-4-5.
	FastModel string

	// SmartModel is used for substantive passes that need more reasoning
	// — future `:proof %` paragraph-level rewrites. Defaults to
	// claude-sonnet-4-6.
	SmartModel string
}

func defaultAIConfig() AIConfig {
	return AIConfig{
		FastModel:  "claude-haiku-4-5",
		SmartModel: "claude-sonnet-4-6",
	}
}

func defaultConfig() Config {
	cfg := Config{MaxWidth: 0, CodeTheme: "catppuccin-mocha", Theme: "dark", AI: defaultAIConfig()}
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
	section := ""
	for _, raw := range strings.Split(string(data), "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			section = strings.TrimSpace(line[1 : len(line)-1])
			continue
		}
		eq := strings.IndexByte(line, '=')
		if eq < 0 {
			continue
		}
		key := strings.TrimSpace(line[:eq])
		value := strings.Trim(strings.TrimSpace(line[eq+1:]), `"'`)
		switch section {
		case "":
			applyTopLevelKey(&cfg, key, value)
		case "ai":
			applyAIKey(&cfg.AI, key, value)
		}
	}
	return cfg
}

func applyTopLevelKey(cfg *Config, key, value string) {
	switch key {
	case "max_width":
		if n, err := strconv.Atoi(value); err == nil && n > 0 {
			cfg.MaxWidth = n
		}
	case "code_max_width":
		if n, err := strconv.Atoi(value); err == nil && n >= 0 {
			cfg.CodeMaxWidth = n
		}
	case "center_above":
		if n, err := strconv.Atoi(value); err == nil && n >= 0 {
			cfg.CenterAbove = n
		}
	case "code_theme":
		if value != "" {
			cfg.CodeTheme = value
		}
	case "theme":
		if value != "" {
			cfg.Theme = value
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

func applyAIKey(ai *AIConfig, key, value string) {
	switch key {
	case "api_key":
		ai.APIKey = value
	case "fast_model":
		if value != "" {
			ai.FastModel = value
		}
	case "smart_model":
		if value != "" {
			ai.SmartModel = value
		}
	}
}

// parseRootList accepts a TOML-style array literal (`["~", "/Users/me/notes"]`)
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
