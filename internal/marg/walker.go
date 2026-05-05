package marg

import (
	"os"
	"os/exec"
	"path/filepath"
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
//
// Any extra directories named in `include_dirs` (e.g. `.claude`) are located
// under the given roots and walked as additional roots so their markdown
// files show up despite the default exclusion rules.
func findMarkdownFiles(roots []string) []string {
	if len(roots) == 0 {
		return nil
	}
	allRoots := append([]string{}, roots...)
	if names := includeDirList(); len(names) > 0 {
		allRoots = append(allRoots, findIncludeRoots(roots, names)...)
	}
	allRoots = uniqueStrings(allRoots)

	if path, err := exec.LookPath("fd"); err == nil {
		if out, err := walkWithFd(path, allRoots); err == nil {
			return out
		}
	}
	if path, err := exec.LookPath("rg"); err == nil {
		if out, err := walkWithRg(path, allRoots); err == nil {
			return out
		}
	}
	var all []string
	for _, r := range allRoots {
		all = append(all, collectMarkdownFiles(r)...)
	}
	return all
}

// findIncludeRoots locates every directory under any of `roots` whose
// basename matches one of `names`. We always use the manual walker here:
// fd's `--hidden` would also descend into unrelated hidden dirs (`.cargo`,
// `.npm`), and we only want hidden dirs that the user explicitly asked for.
func findIncludeRoots(roots, names []string) []string {
	wanted := make(map[string]bool, len(names))
	for _, n := range names {
		wanted[n] = true
	}
	var out []string
	for _, root := range roots {
		walkForIncludeDirs(root, wanted, &out)
	}
	return out
}

// walkForIncludeDirs descends `root` looking for directories whose basename
// is in `wanted`. When one is found, its absolute path goes into out and we
// don't descend further. We only descend through visible, non-noise
// ancestors — the include name itself is allowed to be hidden, but we don't
// want to surface matches buried under unrelated hidden dirs like `.cargo`.
func walkForIncludeDirs(root string, wanted map[string]bool, out *[]string) {
	filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		if err != nil || d == nil || !d.IsDir() {
			return nil
		}
		name := filepath.Base(p)
		if wanted[name] {
			*out = append(*out, p)
			return filepath.SkipDir
		}
		if p != root && (strings.HasPrefix(name, ".") || isIgnoredDir(name)) {
			return filepath.SkipDir
		}
		return nil
	})
}

func uniqueStrings(in []string) []string {
	seen := make(map[string]bool, len(in))
	out := make([]string, 0, len(in))
	for _, s := range in {
		if seen[s] {
			continue
		}
		seen[s] = true
		out = append(out, s)
	}
	return out
}

func walkWithFd(bin string, roots []string) ([]string, error) {
	args := []string{
		"--type", "f",
		"--extension", "md",
		"--extension", "markdown",
	}
	for _, d := range ignoredDirList() {
		args = append(args, "--exclude", d)
	}
	args = append(args, ".")
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
	}
	for _, d := range ignoredDirList() {
		args = append(args, "--glob", "!"+d)
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
