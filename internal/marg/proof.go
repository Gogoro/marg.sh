package marg

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/charmbracelet/lipgloss"
)

// suggestionKind tags a proofreading note as either a mechanical fix
// (spelling, grammar) or a stylistic note (passive voice, wordiness).
// The MVP renders both the same way; future phases may differ.
type suggestionKind int

const (
	kindMechanical suggestionKind = iota
	kindStylistic
)

// suggestion is one proofreading note anchored to a single line of the
// buffer at a given moment. The position is recomputed on every update
// from the underlying proofItem so it tracks the user's edits.
// itemIdx is the index back into the editor's proofItems slice — used
// when accepting or rejecting so we know which underlying item to
// dismiss without searching.
type suggestion struct {
	itemIdx     int
	row         int
	startCol    int
	endCol      int
	original    string
	replacement string
	reason      string
	kind        suggestionKind
}

// proofItem wraps a rawSuggestion with its in-session state. dismissed
// means the user accepted or rejected this item; we keep it in the
// list but skip it during anchoring so it stops appearing in boxes
// without rebuilding the canonical list.
type proofItem struct {
	raw       rawSuggestion
	dismissed bool
}

// rawSuggestion is what the model returns. Anchored to a buffer position
// later, in anchorSuggestions.
type rawSuggestion struct {
	Original    string `json:"original"`
	Replacement string `json:"replacement"`
	Reason      string `json:"reason"`
	Kind        string `json:"kind"`
}

// proofResultMsg is delivered back through Bubble Tea after a `:proof` call
// finishes. The editor reads it via onProofResult.
type proofResultMsg struct {
	suggestions []rawSuggestion
	err         error
}

const proofSystemPrompt = `You are a precise proofreader for technical writing in markdown documents.

Look for two kinds of issues:
- "mechanical": spelling errors, dropped or duplicated words, subject-verb agreement, basic grammar, punctuation.
- "stylistic": passive voice where active is clearer, wordiness, weak verbs, unnecessary hedging, redundancy.

Do NOT suggest:
- Pure rephrasings that don't make the text clearer or more correct.
- Changes inside fenced code blocks, inline code (between backticks), URLs, file paths, or technical jargon.
- Anything that changes the author's voice, tone, or technical meaning.
- Stylistic preferences (Oxford comma, "which" vs "that") unless clearly wrong.

Return ONLY a JSON array of suggestions. No prose before or after. No code fence. No commentary.

Each item has these fields:
- "original": the exact span from the document, copied verbatim, preserving case and punctuation. Must appear verbatim on a single line of the document.
- "replacement": the corrected text. May be empty for advisory-only notes.
- "reason": one short lowercase phrase, no period (e.g. "missing 'the'", "passive voice", "wordy").
- "kind": "mechanical" or "stylistic".

If the same issue appears multiple times, include only the first occurrence.

If the document has no issues, return [].`

// requestProofread calls the Anthropic API with the document text and parses
// the JSON array of suggestions out of the response. The returned suggestions
// are unanchored — the editor anchors them onto the current buffer when the
// result arrives.
//
// `apiKey` from config wins; an empty apiKey falls back to the
// ANTHROPIC_API_KEY env var. `model` is the model ID string from config.
func requestProofread(ctx context.Context, doc, model, apiKey string) ([]rawSuggestion, error) {
	if apiKey == "" {
		apiKey = os.Getenv("ANTHROPIC_API_KEY")
	}
	if apiKey == "" {
		return nil, errors.New("no API key — set [ai] api_key in config or ANTHROPIC_API_KEY")
	}
	if model == "" {
		model = "claude-haiku-4-5"
	}
	client := anthropic.NewClient(option.WithAPIKey(apiKey))
	resp, err := client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     anthropic.Model(model),
		MaxTokens: 4096,
		System: []anthropic.TextBlockParam{{
			Text: proofSystemPrompt,
		}},
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(doc)),
		},
	})
	if err != nil {
		return nil, err
	}
	var text strings.Builder
	for _, block := range resp.Content {
		if t, ok := block.AsAny().(anthropic.TextBlock); ok {
			text.WriteString(t.Text)
		}
	}
	return parseProofJSON(text.String())
}

// parseProofJSON pulls a JSON array out of the model's response. The system
// prompt asks for a bare array, but we tolerate stray prose and ```json fences
// rather than fail noisily on a borderline-malformed response.
func parseProofJSON(s string) ([]rawSuggestion, error) {
	s = strings.TrimSpace(s)
	start := strings.Index(s, "[")
	end := strings.LastIndex(s, "]")
	if start < 0 || end < start {
		return nil, errors.New("no JSON array in proofread response")
	}
	var raws []rawSuggestion
	if err := json.Unmarshal([]byte(s[start:end+1]), &raws); err != nil {
		return nil, fmt.Errorf("parse proofread JSON: %w", err)
	}
	return raws, nil
}

// parseSuggestionKind maps the model's "kind" string onto our enum.
// Anything other than "stylistic" is treated as mechanical.
func parseSuggestionKind(s string) suggestionKind {
	if s == "stylistic" {
		return kindStylistic
	}
	return kindMechanical
}

// renderProofBox builds per-suggestion bordered boxes anchored to the
// visual row where each suggestion's span starts. Boxes never rearrange
// — each one keeps its preferred row. Where boxes overlap, the hovered
// one wins in Z-order: it renders last so its rows overwrite whatever
// non-hovered boxes had drawn beneath, leaving back boxes peeking out
// at the edges of the front one (the "depth" stack).
//
// Bottom-edge awareness: a box whose preferred top would push its
// bottom past the viewport gets shifted up so its bottom border lands
// on the viewport's last row. Top-edge awareness: a box whose anchor
// is above the viewport is hidden — the prose it points at isn't on
// screen, so a floating box would be confusing.
//
// Returns one string per visual row in the viewport, or nil when the
// box can't be shown (no suggestions, or not enough horizontal room).
func (e *editor) renderProofBox(visuals []visualLine, rowCount int) []string {
	if len(e.suggestions) == 0 || rowCount <= 0 {
		return nil
	}
	const (
		minTotalBox = 28
		maxTotalBox = 40
		gapBefore   = 4
		framePad    = 4 // 2 border + 2 internal padding (1 each side)
	)
	available := e.width - e.leftMargin() - e.wrapWidth() - gapBefore
	if available < minTotalBox {
		return nil
	}
	totalBox := maxTotalBox
	if available < totalBox {
		totalBox = available
	}
	contentW := totalBox - framePad
	if contentW < 12 {
		return nil
	}

	hoveredIdx := -1
	for i, s := range e.suggestions {
		if s.row == e.row && e.col >= s.startCol && e.col < s.endCol {
			hoveredIdx = i
			break
		}
	}

	// Place each suggestion at its anchor, bottom-clamped. Order in
	// `placed`: non-hovered in document order, then hovered last so
	// its cells overwrite anything beneath in the render step below.
	var placed []placedBox
	var hoveredPlaced *placedBox
	for i, s := range e.suggestions {
		visIdx := visualIndexForSuggestion(visuals, s)
		if visIdx < 0 {
			continue
		}
		anchor := visIdx - e.scroll
		if anchor < 0 || anchor >= rowCount {
			continue
		}
		isHovered := i == hoveredIdx
		content := buildSingleProofBox(s, contentW, isHovered)
		h := len(content)
		top := anchor
		if top+h > rowCount {
			top = rowCount - h
		}
		if top < 0 {
			top = 0
		}
		box := placedBox{row: top, height: h, content: content}
		if isHovered {
			b := box
			hoveredPlaced = &b
		} else {
			placed = append(placed, box)
		}
	}
	if hoveredPlaced != nil {
		placed = append(placed, *hoveredPlaced)
	}

	out := make([]string, rowCount)
	spacer := strings.Repeat(" ", totalBox)
	for i := range out {
		out[i] = spacer
	}
	for _, b := range placed {
		for i, line := range b.content {
			if b.row+i >= rowCount {
				break
			}
			out[b.row+i] = line
		}
	}
	return out
}

// visualIndexForSuggestion returns the index into visuals where a
// suggestion's span starts, or -1 if it can't be matched (line edited
// after the proofread response was anchored).
func visualIndexForSuggestion(visuals []visualLine, s suggestion) int {
	for i, v := range visuals {
		if v.row != s.row {
			continue
		}
		if s.startCol >= v.startCol && s.startCol < v.endCol {
			return i
		}
		// Empty-line anchor: span starts at col 0 on a row with no chars.
		if v.startCol == 0 && v.endCol == 0 && s.startCol == 0 {
			return i
		}
	}
	return -1
}

// placedBox is a sidebar box that has been positioned at a viewport row.
// `content` is height-many pre-rendered lines including borders.
type placedBox struct {
	row, height int
	content     []string
}

// buildSingleProofBox renders one suggestion as a stand-alone bordered
// box. The heading is painted in the same color as the border so the
// box reads as a single colored identity. Hovered boxes are in the
// accent color with a brighter reason; non-hovered boxes use a
// kind-based color (mechanical = warm, stylistic = cool) with a muted
// italic reason so the type reads at a glance and hovered boxes
// dominate the eye.
func buildSingleProofBox(s suggestion, contentW int, hovered bool) []string {
	var border, head, reason lipgloss.Color
	if hovered {
		border = colorAccent
		head = colorAccent
		reason = colorFg
	} else {
		border = kindBorderColor(s.kind)
		head = border
		reason = colorMuted
	}

	headStyle := lipgloss.NewStyle().Foreground(head).Bold(true)
	reasonStyle := lipgloss.NewStyle().Foreground(reason).Italic(true)

	var lines []string
	pad := func(line string) string {
		w := lipgloss.Width(line)
		if w >= contentW {
			return line
		}
		return line + strings.Repeat(" ", contentW-w)
	}

	headText := "→ " + s.replacement
	showReason := s.replacement != "" && s.reason != ""
	if s.replacement == "" {
		headText = "· " + s.reason
	}
	for _, w := range wrapForBox(headText, contentW) {
		lines = append(lines, pad(headStyle.Render(w)))
	}
	if showReason {
		for _, w := range wrapForBox("  "+s.reason, contentW) {
			lines = append(lines, pad(reasonStyle.Render(w)))
		}
	}

	body := strings.Join(lines, "\n")
	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(border).
		Padding(0, 1)
	rendered := style.Render(body)
	return strings.Split(rendered, "\n")
}

// kindBorderColor picks a non-hovered border color from the active
// theme's heading palette so we don't add new color slots: mechanical
// uses the H3 (warm yellow / amber across themes), stylistic uses the
// H6 (cool violet / lavender). The two read distinctly on every theme
// while staying within the existing palette.
func kindBorderColor(k suggestionKind) lipgloss.Color {
	if k == kindStylistic {
		return colorHeadings[5]
	}
	return colorHeadings[2]
}

// wrapForBox does word-aware wrapping at a column width. Falls back to a
// hard break when there's no space to break on.
func wrapForBox(s string, width int) []string {
	if width <= 0 {
		return []string{s}
	}
	runes := []rune(s)
	if len(runes) <= width {
		return []string{string(runes)}
	}
	var out []string
	for len(runes) > width {
		brk := -1
		for i := width; i > 0; i-- {
			if runes[i] == ' ' {
				brk = i
				break
			}
		}
		if brk < 0 {
			out = append(out, string(runes[:width]))
			runes = runes[width:]
		} else {
			out = append(out, string(runes[:brk]))
			runes = runes[brk+1:]
		}
	}
	if len(runes) > 0 {
		out = append(out, string(runes))
	}
	return out
}

// anchorSuggestions resolves each non-dismissed proofItem to a buffer
// position by searching for its `original` text line by line. First
// matching line wins. An item whose `original` can't be found is
// silently dropped from this anchored view — usually because the user
// edited that line, or because anchoring runs after an accept that
// replaced the original text.
//
// This is called on every editor update() against the live buffer so
// that boxes follow inserts, deletes, and paste operations rather than
// freezing at the row they had when :proof returned.
func anchorSuggestions(buf *buffer, items []proofItem) []suggestion {
	var out []suggestion
	for i, it := range items {
		if it.dismissed {
			continue
		}
		r := it.raw
		if r.Original == "" {
			continue
		}
		for row := 0; row < buf.lineCount(); row++ {
			line := string(buf.line(row))
			idx := strings.Index(line, r.Original)
			if idx < 0 {
				continue
			}
			startCol := len([]rune(line[:idx]))
			endCol := startCol + len([]rune(r.Original))
			out = append(out, suggestion{
				itemIdx:     i,
				row:         row,
				startCol:    startCol,
				endCol:      endCol,
				original:    r.Original,
				replacement: r.Replacement,
				reason:      r.Reason,
				kind:        parseSuggestionKind(r.Kind),
			})
			break
		}
	}
	return out
}
