package marg

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// chatMessage is one turn in the inline AI conversation. Roles match the
// Anthropic API: "user" or "assistant".
type chatMessage struct {
	role    string
	content string
}

// chat is the AI conversation overlay opened from a visual-mode selection.
// It keeps the selected passage and the surrounding document frozen at open
// time so follow-up edits to the buffer don't quietly change what the model
// is reasoning about.
type chat struct {
	width  int
	height int

	document  string
	selection string
	filepath  string

	messages []chatMessage
	input    string

	sending bool
	err     string

	ai AIConfig

	cancelled bool

	// scroll counts rows lifted off the bottom of the transcript window.
	// 0 means pinned to the tail (latest message visible). Increases as
	// the user scrolls up to read older messages. Snaps back to 0 every
	// time a new message lands so a fresh reply is never hidden.
	scroll int
}

func newChat(ai AIConfig, document, selection, filepath string) chat {
	return chat{
		ai:        ai,
		document:  document,
		selection: selection,
		filepath:  filepath,
	}
}

func (c *chat) resize(w, h int) {
	c.width = w
	c.height = h
}

func (c chat) update(msg tea.KeyMsg) (chat, tea.Cmd) {
	key := msg.String()

	// Scroll keys work even while a request is in flight so the user can
	// re-read earlier turns while waiting on the reply.
	switch key {
	case "up":
		c.scroll++
		c.clampScroll()
		return c, nil
	case "down":
		c.scroll--
		c.clampScroll()
		return c, nil
	case "ctrl+u":
		c.scroll += c.transcriptHeight() / 2
		c.clampScroll()
		return c, nil
	case "ctrl+d":
		c.scroll -= c.transcriptHeight() / 2
		c.clampScroll()
		return c, nil
	case "pgup":
		c.scroll += c.transcriptHeight()
		c.clampScroll()
		return c, nil
	case "pgdown":
		c.scroll -= c.transcriptHeight()
		c.clampScroll()
		return c, nil
	}

	if c.sending {
		if key == "esc" {
			c.cancelled = true
		}
		return c, nil
	}
	switch key {
	case "esc":
		c.cancelled = true
	case "enter":
		text := strings.TrimSpace(c.input)
		if text == "" {
			return c, nil
		}
		c.messages = append(c.messages, chatMessage{role: "user", content: text})
		c.input = ""
		c.err = ""
		c.sending = true
		c.scroll = 0
		return c, c.sendCmd()
	case "backspace":
		if len(c.input) > 0 {
			r := []rune(c.input)
			c.input = string(r[:len(r)-1])
		}
	default:
		if len(msg.Runes) > 0 {
			c.input += string(msg.Runes)
		}
	}
	return c, nil
}

// sendCmd snapshots the conversation and dispatches the API call as a
// tea.Cmd so the UI stays responsive while the model is thinking.
func (c chat) sendCmd() tea.Cmd {
	history := make([]chatMessage, len(c.messages))
	copy(history, c.messages)
	doc := c.document
	sel := c.selection
	fp := c.filepath
	ai := c.ai
	return func() tea.Msg {
		reply, err := requestChat(context.Background(), ai, doc, sel, fp, history)
		return chatResultMsg{reply: reply, err: err}
	}
}

type chatResultMsg struct {
	reply string
	err   error
}

func (c chat) onResult(m chatResultMsg) chat {
	c.sending = false
	if m.err != nil {
		c.err = m.err.Error()
		return c
	}
	c.messages = append(c.messages, chatMessage{role: "assistant", content: strings.TrimSpace(m.reply)})
	c.scroll = 0
	return c
}

const chatSystemPrompt = `You are a writing collaborator helping the author think about a passage from their markdown document.

The full document is provided in <document>. The selected passage to focus on is in <selection>. Use the surrounding document for context, but center every reply on the selection.

Be direct, concrete, and concise. Speak like a thoughtful editor or peer reviewer, not a generic chatbot. No filler openers ("Great question!"). No bullet-listed disclaimers. Match the author's register and don't pad.`

func requestChat(ctx context.Context, ai AIConfig, document, selection, filepath string, history []chatMessage) (string, error) {
	apiKey := ai.APIKey
	if apiKey == "" {
		apiKey = os.Getenv("ANTHROPIC_API_KEY")
	}
	if apiKey == "" {
		return "", errors.New("no API key — set [ai] api_key in config or ANTHROPIC_API_KEY")
	}
	model := ai.SmartModel
	if model == "" {
		model = "claude-sonnet-4-6"
	}
	client := anthropic.NewClient(option.WithAPIKey(apiKey))

	// The first user turn carries the document + selection envelope. Every
	// follow-up turn is plain text so we don't re-send the document on each
	// round trip (cheaper and keeps the prompt cache happy).
	var firstTurn strings.Builder
	if filepath != "" {
		firstTurn.WriteString("<filepath>")
		firstTurn.WriteString(filepath)
		firstTurn.WriteString("</filepath>\n")
	}
	firstTurn.WriteString("<document>\n")
	firstTurn.WriteString(document)
	firstTurn.WriteString("\n</document>\n\n<selection>\n")
	firstTurn.WriteString(selection)
	firstTurn.WriteString("\n</selection>\n\n<question>\n")
	if len(history) > 0 {
		firstTurn.WriteString(history[0].content)
	}
	firstTurn.WriteString("\n</question>")

	var messages []anthropic.MessageParam
	for i, m := range history {
		text := m.content
		if i == 0 && m.role == "user" {
			text = firstTurn.String()
		}
		switch m.role {
		case "user":
			messages = append(messages, anthropic.NewUserMessage(anthropic.NewTextBlock(text)))
		case "assistant":
			messages = append(messages, anthropic.NewAssistantMessage(anthropic.NewTextBlock(text)))
		}
	}

	resp, err := client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     anthropic.Model(model),
		MaxTokens: 2048,
		System: []anthropic.TextBlockParam{{
			Text: chatSystemPrompt,
		}},
		Messages: messages,
	})
	if err != nil {
		return "", err
	}
	var out strings.Builder
	for _, block := range resp.Content {
		if t, ok := block.AsAny().(anthropic.TextBlock); ok {
			out.WriteString(t.Text)
		}
	}
	return out.String(), nil
}

// dims computes the overlay box dimensions and the transcript window size
// from the current terminal size. Centralized so update() can reason about
// page-size scrolling using the same numbers overlay() will render with.
func (c *chat) dims() (boxW, boxH, contentW, transcriptH int) {
	boxW = c.width * 7 / 10
	if boxW < 50 {
		boxW = 50
	}
	if boxW > c.width-4 {
		boxW = c.width - 4
	}
	boxH = c.height * 7 / 10
	if boxH < 14 {
		boxH = 14
	}
	if boxH > c.height-2 {
		boxH = c.height - 2
	}
	contentW = boxW - 2
	// Subtract: title (1) + separator (1) + separator (1) + input (1) = 4
	// rows of chrome, leaving the rest for the transcript.
	transcriptH = boxH - 4
	if transcriptH < 3 {
		transcriptH = 3
	}
	return
}

func (c *chat) transcriptHeight() int {
	_, _, _, h := c.dims()
	return h
}

// clampScroll keeps c.scroll within [0, total-transcriptH]. 0 means pinned
// to the tail; max means the first row of the transcript is at the top of
// the viewport.
func (c *chat) clampScroll() {
	_, _, contentW, transcriptH := c.dims()
	total := len(c.buildRows(contentW))
	maxScroll := total - transcriptH
	if maxScroll < 0 {
		maxScroll = 0
	}
	if c.scroll > maxScroll {
		c.scroll = maxScroll
	}
	if c.scroll < 0 {
		c.scroll = 0
	}
}

// overlay paints the chat box centered on top of the editor view. Layout:
// title row, separator, transcript window, separator, input row.
func (c *chat) overlay(below string) string {
	if c.width == 0 || c.height == 0 {
		return below
	}
	boxW, _, contentW, transcriptH := c.dims()

	title := "ask claude about selection"
	if c.filepath != "" {
		title += "  " + filepath.Base(c.filepath)
	}
	titleStyled := lipgloss.NewStyle().Foreground(colorAccent).Bold(true).Render(padRight(title, contentW))

	sep := lipgloss.NewStyle().Foreground(colorMuted).Render(strings.Repeat("─", contentW))

	transcript := c.renderTranscript(contentW, transcriptH)

	var promptRow string
	switch {
	case c.sending:
		promptRow = lipgloss.NewStyle().Foreground(colorMuted).Italic(true).Render(padRight("thinking…", contentW))
	case c.err != "":
		promptRow = lipgloss.NewStyle().Foreground(colorWarn).Render(padRight("error: "+c.err, contentW))
	default:
		promptRow = padRight("› "+c.input+"_", contentW)
	}

	body := titleStyled + "\n" + sep + "\n" + strings.Join(transcript, "\n") + "\n" + sep + "\n" + promptRow
	box := stylePickerBox.Width(boxW).Render(body)

	return placeOver(below, box, c.width, c.height)
}

// buildRows produces the full transcript as visual rows wrapped to
// contentW. The result is sliced by renderTranscript according to the
// scroll offset, and used by clampScroll to know how far up the user is
// allowed to scroll.
func (c *chat) buildRows(contentW int) []string {
	var rows []string

	if len(c.messages) == 0 {
		hint := "selection: " + previewSelection(c.selection, contentW-len("selection: "))
		rows = append(rows, lipgloss.NewStyle().Foreground(colorMuted).Italic(true).Render(padRight(hint, contentW)))
		rows = append(rows, padRight("", contentW))
		rows = append(rows, lipgloss.NewStyle().Foreground(colorMuted).Render(padRight("type your question and press enter — esc to close", contentW)))
		return rows
	}

	youStyle := lipgloss.NewStyle().Foreground(colorAccent).Bold(true)
	claudeStyle := lipgloss.NewStyle().Foreground(colorBoldFg).Bold(true)

	for i, m := range c.messages {
		if i > 0 {
			rows = append(rows, padRight("", contentW))
		}
		var label string
		var labelStyle lipgloss.Style
		if m.role == "user" {
			label = "you"
			labelStyle = youStyle
		} else {
			label = "claude"
			labelStyle = claudeStyle
		}
		rows = append(rows, padRight(labelStyle.Render(label), contentW))
		for _, paragraph := range strings.Split(m.content, "\n") {
			if paragraph == "" {
				rows = append(rows, padRight("", contentW))
				continue
			}
			for _, line := range wrapForBox(paragraph, contentW) {
				rows = append(rows, padRight(line, contentW))
			}
		}
	}
	return rows
}

// renderTranscript returns exactly rowsAvail rows from the transcript,
// honoring c.scroll. scroll==0 pins the bottom of the transcript to the
// bottom of the window; larger values lift the window up so older
// messages come into view.
func (c *chat) renderTranscript(contentW, rowsAvail int) []string {
	rows := c.buildRows(contentW)

	if len(rows) <= rowsAvail {
		for len(rows) < rowsAvail {
			rows = append(rows, padRight("", contentW))
		}
		return rows
	}

	maxScroll := len(rows) - rowsAvail
	if c.scroll > maxScroll {
		c.scroll = maxScroll
	}
	if c.scroll < 0 {
		c.scroll = 0
	}
	end := len(rows) - c.scroll
	start := end - rowsAvail
	return rows[start:end]
}

// previewSelection returns a one-line preview of the selected passage,
// collapsing whitespace and truncating to fit. Used in the empty-state hint
// so users see what they're about to ask about.
func previewSelection(s string, width int) string {
	if width < 4 {
		width = 4
	}
	flat := strings.Join(strings.Fields(s), " ")
	r := []rune(flat)
	if len(r) <= width {
		return flat
	}
	return string(r[:width-1]) + "…"
}
