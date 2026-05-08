package marg

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"regexp"
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

	// streamCh carries chunks from the in-flight streaming request. cancel
	// kills the goroutine when the overlay is dismissed mid-flight so the
	// reply is never delivered after close. Both are nil while idle.
	streamCh chan chatStreamMsg
	cancel   context.CancelFunc

	// armEdit is true when the user pressed ctrl+e to mark the *next*
	// submission as an edit-suggestion request. pendingEdit captures that
	// armed state at submit time so the in-flight reply is parsed as
	// structured edits when the stream ends.
	armEdit     bool
	pendingEdit bool

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
	case "ctrl+e":
		c.armEdit = !c.armEdit
		return c, nil
	case "enter":
		text := strings.TrimSpace(c.input)
		if text == "" {
			return c, nil
		}
		c.pendingEdit = c.armEdit
		c.armEdit = false
		c.messages = append(c.messages, chatMessage{role: "user", content: text})
		c.input = ""
		c.err = ""
		c.sending = true
		c.scroll = 0
		// Push an empty assistant placeholder so streaming chunks have
		// somewhere to land as they arrive.
		c.messages = append(c.messages, chatMessage{role: "assistant", content: ""})
		return c, c.startStreaming()
	case "alt+enter", "shift+enter", "ctrl+j":
		// Newline within the input. Three bindings because terminal
		// reporting varies — alt+enter is standard in iTerm2, shift+enter
		// works in some terminals, ctrl+j is the classic ASCII LF fallback.
		c.input += "\n"
	case "ctrl+y":
		// Yank the most recent assistant reply to the system clipboard so
		// the user can paste it back into the doc (or anywhere else)
		// without manually selecting and copying.
		reply := c.lastAssistantReply()
		if reply == "" {
			c.err = "no reply to yank"
			return c, nil
		}
		if err := setSystemClipboard(reply); err != nil {
			c.err = "clipboard: " + err.Error()
			return c, nil
		}
		c.err = "→ clipboard"
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

// startStreaming opens a streaming request and returns a tea.Cmd that
// reads the first chunk off the channel. Each subsequent chunk is fetched
// by chaining new readCmd calls from onStream until the stream ends.
func (c *chat) startStreaming() tea.Cmd {
	history := make([]chatMessage, len(c.messages)-1) // exclude the empty assistant placeholder
	copy(history, c.messages[:len(c.messages)-1])
	doc := c.document
	sel := c.selection
	fp := c.filepath
	ai := c.ai
	editMode := c.pendingEdit

	ctx, cancel := context.WithCancel(context.Background())
	ch := make(chan chatStreamMsg, 16)
	c.cancel = cancel
	c.streamCh = ch

	go func() {
		defer close(ch)
		streamChat(ctx, ai, doc, sel, fp, history, editMode, ch)
	}()
	return readChatStream(ch)
}

// chatStreamMsg is one event from the streaming pipeline. chunk is a
// piece of the assistant's reply; end signals the stream is finished;
// err carries a transport or model error.
type chatStreamMsg struct {
	chunk string
	err   error
	end   bool
}

func readChatStream(ch <-chan chatStreamMsg) tea.Cmd {
	return func() tea.Msg {
		m, ok := <-ch
		if !ok {
			return chatStreamMsg{end: true}
		}
		return m
	}
}

// onStream handles a single streaming event. It appends the chunk to the
// trailing assistant message, or finalizes / records the error on end.
// Returns a Cmd that reads the next event, or nil when the stream is done.
func (c chat) onStream(m chatStreamMsg) (chat, tea.Cmd) {
	if m.err != nil {
		c.err = m.err.Error()
		c.sending = false
		c.cancel = nil
		c.streamCh = nil
		// Drop the empty trailing placeholder if no chunks landed.
		if n := len(c.messages); n > 0 && c.messages[n-1].role == "assistant" && c.messages[n-1].content == "" {
			c.messages = c.messages[:n-1]
		}
		return c, nil
	}
	if m.end {
		c.sending = false
		c.cancel = nil
		c.streamCh = nil
		c.scroll = 0
		if c.pendingEdit {
			c.pendingEdit = false
			text := ""
			if n := len(c.messages); n > 0 && c.messages[n-1].role == "assistant" {
				text = c.messages[n-1].content
				// Strip the raw JSON reply from the transcript — we'll
				// replace it with a status line once the parse lands.
				c.messages = c.messages[:n-1]
			}
			return c, dispatchEditsCmd(text)
		}
		if n := len(c.messages); n > 0 && c.messages[n-1].role == "assistant" {
			c.messages[n-1].content = strings.TrimSpace(c.messages[n-1].content)
			if c.messages[n-1].content == "" {
				c.messages = c.messages[:n-1]
			}
		}
		return c, nil
	}
	if n := len(c.messages); n > 0 && c.messages[n-1].role == "assistant" {
		c.messages[n-1].content += m.chunk
	}
	return c, readChatStream(c.streamCh)
}

const chatSystemPrompt = `You are a writing collaborator helping the author think about a passage from their markdown document.

The full document is provided in <document>. The selected passage to focus on is in <selection>. Use the surrounding document for context, but center every reply on the selection.

Be direct, concrete, and concise. Speak like a thoughtful editor or peer reviewer, not a generic chatbot. No filler openers ("Great question!"). No bullet-listed disclaimers. Match the author's register and don't pad.`

const chatEditPrompt = `You are an editing collaborator. The author has selected a passage from their markdown document and wants concrete edits to it.

Return ONLY a JSON array of suggested edits. No prose before or after, no code fence, no commentary.

Each item has:
- "original": the exact span from the document, copied verbatim, preserving case and punctuation. Must appear verbatim on a single line of the document.
- "replacement": the proposed text. May be empty for advisory-only notes.
- "reason": one short lowercase phrase, no period (e.g. "tighter phrasing", "redundant", "wrong word").
- "kind": "mechanical" (spelling/grammar) or "stylistic" (voice/wordiness).

Focus your edits on the <selection>. Only emit edits the author could plausibly accept blindly — no rewrites that change meaning, no rephrasings that aren't clearly better. If the user asks a specific question, answer it through your choice of edits.

If you can't find good edits, return [].`

// dispatchEditsCmd parses the assistant's JSON reply asynchronously and
// emits a chatEditsMsg for the app to route into the editor's proof
// pipeline. Done as a Cmd so the parsing happens off the Update goroutine.
func dispatchEditsCmd(text string) tea.Cmd {
	return func() tea.Msg {
		raws, err := parseProofJSON(text)
		return chatEditsMsg{raws: raws, err: err}
	}
}

// chatEditsMsg is the result of an edit-mode round-trip: structured
// suggestions ready to anchor onto the buffer (raws) or a parse error.
type chatEditsMsg struct {
	raws []rawSuggestion
	err  error
}

// streamChat opens a streaming request to the Anthropic API and pushes
// text-delta chunks onto ch as they arrive. ch is closed by the caller's
// defer; on error or context cancellation, this sends a terminal event
// and returns. The first user turn is wrapped in <document> / <selection>
// / <question> XML so the model knows the context envelope; later turns
// are plain text.
func streamChat(ctx context.Context, ai AIConfig, document, selection, filepath string, history []chatMessage, editMode bool, ch chan<- chatStreamMsg) {
	send := func(m chatStreamMsg) {
		select {
		case ch <- m:
		case <-ctx.Done():
		}
	}

	apiKey := ai.APIKey
	if apiKey == "" {
		apiKey = os.Getenv("ANTHROPIC_API_KEY")
	}
	if apiKey == "" {
		send(chatStreamMsg{err: errors.New("no API key — set [ai] api_key in config or ANTHROPIC_API_KEY"), end: true})
		return
	}
	model := ai.SmartModel
	if model == "" {
		model = "claude-sonnet-4-6"
	}
	client := anthropic.NewClient(option.WithAPIKey(apiKey))

	var question string
	if len(history) > 0 {
		question = history[0].content
	}
	attachments := loadMentionedAttachments(question, filepath)

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
	firstTurn.WriteString("\n</selection>\n")
	if attachments != "" {
		firstTurn.WriteString("\n<attachments>\n")
		firstTurn.WriteString(attachments)
		firstTurn.WriteString("</attachments>\n")
	}
	firstTurn.WriteString("\n<question>\n")
	firstTurn.WriteString(question)
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

	systemPrompt := chatSystemPrompt
	if editMode {
		systemPrompt = chatEditPrompt
	}
	stream := client.Messages.NewStreaming(ctx, anthropic.MessageNewParams{
		Model:     anthropic.Model(model),
		MaxTokens: 2048,
		System: []anthropic.TextBlockParam{{
			Text: systemPrompt,
		}},
		Messages: messages,
	})
	defer stream.Close()

	for stream.Next() {
		evt := stream.Current()
		if delta, ok := evt.AsAny().(anthropic.ContentBlockDeltaEvent); ok {
			if td, ok := delta.Delta.AsAny().(anthropic.TextDelta); ok && td.Text != "" {
				send(chatStreamMsg{chunk: td.Text})
			}
		}
		if ctx.Err() != nil {
			return
		}
	}
	if err := stream.Err(); err != nil && ctx.Err() == nil {
		send(chatStreamMsg{err: err, end: true})
		return
	}
	send(chatStreamMsg{end: true})
}

// dims computes the overlay box dimensions and the transcript window size
// from the current terminal size. Centralized so update() can reason about
// page-size scrolling using the same numbers overlay() will render with.
// The input area grows up to maxInputRows lines as the user types
// shift+enter; the transcript shrinks accordingly.
func (c *chat) dims() (boxW, boxH, contentW, transcriptH, inputRows int) {
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
	inputRows = c.inputRows()
	// Subtract: title (1) + separator (1) + separator (1) + input (n) =
	// 3+n rows of chrome, leaving the rest for the transcript.
	transcriptH = boxH - 3 - inputRows
	if transcriptH < 3 {
		transcriptH = 3
	}
	return
}

// lastAssistantReply returns the content of the most recent assistant
// message, or "" if there isn't one yet.
func (c *chat) lastAssistantReply() string {
	for i := len(c.messages) - 1; i >= 0; i-- {
		if c.messages[i].role == "assistant" && c.messages[i].content != "" {
			return c.messages[i].content
		}
	}
	return ""
}

// inputRows returns the number of rows the input area should occupy given
// the current input text. Caps at 5 to keep the transcript usable.
func (c *chat) inputRows() int {
	const maxInputRows = 5
	rows := strings.Count(c.input, "\n") + 1
	if rows > maxInputRows {
		return maxInputRows
	}
	if rows < 1 {
		return 1
	}
	return rows
}

func (c *chat) transcriptHeight() int {
	_, _, _, h, _ := c.dims()
	return h
}

// clampScroll keeps c.scroll within [0, total-transcriptH]. 0 means pinned
// to the tail; max means the first row of the transcript is at the top of
// the viewport.
func (c *chat) clampScroll() {
	_, _, contentW, transcriptH, _ := c.dims()
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
	boxW, _, contentW, transcriptH, inputRows := c.dims()

	title := "ask claude about selection"
	if c.filepath != "" {
		title += "  " + filepath.Base(c.filepath)
	}
	titleStyled := lipgloss.NewStyle().Foreground(colorAccent).Bold(true).Render(padRight(title, contentW))

	sep := lipgloss.NewStyle().Foreground(colorMuted).Render(strings.Repeat("─", contentW))

	transcript := c.renderTranscript(contentW, transcriptH)

	promptRows := c.renderInputRows(contentW, inputRows)

	body := titleStyled + "\n" + sep + "\n" + strings.Join(transcript, "\n") + "\n" + sep + "\n" + strings.Join(promptRows, "\n")
	box := stylePickerBox.Width(boxW).Render(body)

	return placeOver(below, box, c.width, c.height)
}

// renderInputRows builds the rows of the input area. Single-line input
// matches the old behavior (`› text_`); multi-line shows each line with
// the prompt glyph on the first row and continuation indent on the rest,
// followed by a cursor `_` on the last line. Sending / error states still
// take over the whole input area as before.
func (c *chat) renderInputRows(contentW, rows int) []string {
	if c.sending {
		out := make([]string, rows)
		out[0] = lipgloss.NewStyle().Foreground(colorMuted).Italic(true).Render(padRight("thinking…", contentW))
		for i := 1; i < rows; i++ {
			out[i] = padRight("", contentW)
		}
		return out
	}
	if c.err != "" {
		out := make([]string, rows)
		out[0] = lipgloss.NewStyle().Foreground(colorWarn).Render(padRight("error: "+c.err, contentW))
		for i := 1; i < rows; i++ {
			out[i] = padRight("", contentW)
		}
		return out
	}
	lines := strings.Split(c.input, "\n")
	out := make([]string, rows)
	for i := 0; i < rows; i++ {
		var line string
		if i < len(lines) {
			line = lines[i]
		}
		prefix := "  "
		if i == 0 {
			if c.armEdit {
				prefix = lipgloss.NewStyle().Foreground(colorAccent).Bold(true).Render("✎ ")
			} else {
				prefix = "› "
			}
		}
		row := prefix + line
		if i == len(lines)-1 || (i == rows-1 && len(lines) > rows) {
			row += "_"
		}
		out[i] = padRight(row, contentW)
	}
	return out
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

// mentionRegex matches `@path/to/file.md` tokens in the chat input. We
// allow word characters, dots, dashes, slashes, and tildes — enough for
// most reasonable filenames without sweeping in punctuation that follows
// the path. To attach a literal `@` without resolving a file, prefix it
// with whitespace and an unrecognized character.
var mentionRegex = regexp.MustCompile(`@([\w\-./~]+)`)

// loadMentionedAttachments scans the question for `@file` tokens, resolves
// each one relative to the current document's directory (or expands
// `~/...`), reads the contents, and returns one big <attachment> block per
// resolved file. Missing files are silently skipped — the user might be
// using `@something` as plain text.
func loadMentionedAttachments(question, currentFile string) string {
	matches := mentionRegex.FindAllStringSubmatch(question, -1)
	if len(matches) == 0 {
		return ""
	}
	baseDir := ""
	if currentFile != "" {
		baseDir = filepath.Dir(currentFile)
	}
	home, _ := os.UserHomeDir()
	seen := map[string]bool{}
	var out strings.Builder
	for _, m := range matches {
		raw := m[1]
		var resolved string
		switch {
		case strings.HasPrefix(raw, "~/") && home != "":
			resolved = filepath.Join(home, raw[2:])
		case filepath.IsAbs(raw):
			resolved = raw
		case baseDir != "":
			resolved = filepath.Join(baseDir, raw)
		default:
			resolved = raw
		}
		if seen[resolved] {
			continue
		}
		seen[resolved] = true
		data, err := os.ReadFile(resolved)
		if err != nil {
			continue
		}
		out.WriteString(`<attachment path="`)
		out.WriteString(resolved)
		out.WriteString("\">\n")
		out.Write(data)
		if len(data) > 0 && data[len(data)-1] != '\n' {
			out.WriteByte('\n')
		}
		out.WriteString("</attachment>\n")
	}
	return out.String()
}
