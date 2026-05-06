package marg

// defaultIgnoreDirs lists the directory basenames marg never descends into
// when looking for markdown files. These show up across most developer
// machines and contain no notes worth opening.
var defaultIgnoreDirs = []string{
	// language toolchains / package caches
	"node_modules",
	"vendor",
	"go", // GOPATH root: pkg/mod is full of vendored READMEs
	"Pods",
	"Carthage",
	"target",
	"build",
	"dist",
	"DerivedData",
	"coverage",
	// macOS noise
	"Library",
	"Applications",
}

// ignoreDirs is the active set: defaults merged with the user's additions
// from `ignore_dirs` in config. Updated once at startup.
var ignoreDirs = ignoreSetFrom(defaultIgnoreDirs, nil)

// includeDirs lists directory basenames that should ALWAYS be searched even
// when they would normally be filtered out (e.g. dot-prefixed dirs like
// `.claude` or `.obsidian`). Populated from `include_dirs` in config.
var includeDirs []string

// applyDirConfig is called by Run after the config loads to apply both the
// user's ignore additions and their include-list overrides everywhere.
func applyDirConfig(extraIgnore, extraInclude []string) {
	ignoreDirs = ignoreSetFrom(defaultIgnoreDirs, extraIgnore)
	includeDirs = append([]string{}, extraInclude...)
}

func ignoreSetFrom(base, extra []string) map[string]bool {
	out := make(map[string]bool, len(base)+len(extra))
	for _, d := range base {
		if d != "" {
			out[d] = true
		}
	}
	for _, d := range extra {
		if d != "" {
			out[d] = true
		}
	}
	return out
}

func isIgnoredDir(name string) bool {
	return ignoreDirs[name]
}

// ignoredDirList returns the current ignore set as a slice — handy when we
// need to pass it as `--exclude` flags to fd / rg.
func ignoredDirList() []string {
	out := make([]string, 0, len(ignoreDirs))
	for d := range ignoreDirs {
		out = append(out, d)
	}
	return out
}

func includeDirList() []string {
	return includeDirs
}
