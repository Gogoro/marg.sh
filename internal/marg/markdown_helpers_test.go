package marg

import "testing"

func TestParseListLineBullet(t *testing.T) {
	info, ok := parseListLine([]rune("- foo bar"))
	if !ok {
		t.Fatal("expected list match")
	}
	if info.prefixRunes != 2 {
		t.Errorf("prefixRunes = %d, want 2", info.prefixRunes)
	}
	if info.nextPrefix != "- " {
		t.Errorf("nextPrefix = %q, want %q", info.nextPrefix, "- ")
	}
}

func TestParseListLineCheckboxUnchecked(t *testing.T) {
	info, ok := parseListLine([]rune("- [ ] make tea"))
	if !ok {
		t.Fatal("expected list match")
	}
	if info.prefixRunes != 6 {
		t.Errorf("prefixRunes = %d, want 6", info.prefixRunes)
	}
	if info.nextPrefix != "- [ ] " {
		t.Errorf("nextPrefix = %q, want %q", info.nextPrefix, "- [ ] ")
	}
}

func TestParseListLineCheckboxChecked(t *testing.T) {
	info, ok := parseListLine([]rune("- [x] done thing"))
	if !ok {
		t.Fatal("expected list match")
	}
	if info.prefixRunes != 6 {
		t.Errorf("prefixRunes = %d, want 6", info.prefixRunes)
	}
	if info.nextPrefix != "- [ ] " {
		t.Errorf("nextPrefix = %q, want %q (always unchecked next)", info.nextPrefix, "- [ ] ")
	}
}

func TestParseListLineCheckboxIndented(t *testing.T) {
	info, ok := parseListLine([]rune("    - [X] nested"))
	if !ok {
		t.Fatal("expected list match")
	}
	if info.prefixRunes != 10 {
		t.Errorf("prefixRunes = %d, want 10", info.prefixRunes)
	}
	if info.nextPrefix != "    - [ ] " {
		t.Errorf("nextPrefix = %q, want %q", info.nextPrefix, "    - [ ] ")
	}
}

func TestParseListLineCheckboxAlternateBullets(t *testing.T) {
	for _, c := range []string{"*", "+"} {
		info, ok := parseListLine([]rune(c + " [ ] task"))
		if !ok {
			t.Fatalf("expected list match for bullet %q", c)
		}
		if info.prefixRunes != 6 {
			t.Errorf("bullet %q: prefixRunes = %d, want 6", c, info.prefixRunes)
		}
		want := c + " [ ] "
		if info.nextPrefix != want {
			t.Errorf("bullet %q: nextPrefix = %q, want %q", c, info.nextPrefix, want)
		}
	}
}

func TestParseListLineNotACheckbox(t *testing.T) {
	info, ok := parseListLine([]rune("- [foo] bar"))
	if !ok {
		t.Fatal("expected list match (still a bullet)")
	}
	if info.prefixRunes != 2 {
		t.Errorf("prefixRunes = %d, want 2 (no checkbox detected)", info.prefixRunes)
	}
}
