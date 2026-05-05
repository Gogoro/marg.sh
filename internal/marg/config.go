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
}

func defaultConfig() Config {
	return Config{MaxWidth: 0}
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
		}
	}
	return cfg
}

func configPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "marg", "config.toml"), nil
}
