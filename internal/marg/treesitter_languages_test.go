package marg

import "testing"

// TestAllTSLanguagesLoad ensures every language we register actually gets
// its query compiled. A non-nil result means loadTSLanguage's ReadFile +
// NewQuery path succeeded for that grammar; a nil entry would mean we'd
// silently fall back to Chroma at runtime, which we don't want for
// languages we explicitly ship.
func TestAllTSLanguagesLoad(t *testing.T) {
	want := []string{
		"sql", "go", "javascript", "typescript", "rust", "python",
		"bash", "yaml", "toml", "lua", "html", "css", "dockerfile",
	}
	got := make(map[string]bool)
	for _, l := range tsLanguages {
		if l != nil {
			got[l.name] = true
		}
	}
	for _, name := range want {
		if !got[name] {
			t.Errorf("language %q failed to load (query parse error?)", name)
		}
	}
}
