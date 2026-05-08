package marg

import (
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

// Run is the entrypoint called from main. Parses args, builds the initial
// model, and starts the Bubble Tea program.
func Run(args []string) error {
	target, err := parseArgs(args)
	if err != nil {
		return err
	}

	// Force truecolor when the terminal advertises it via COLORTERM. Inside
	// tmux, lipgloss/termenv otherwise downsamples to the 256-color palette
	// because TERM is `tmux-256color`, which makes themes like monokai look
	// muddy.
	colorterm := strings.ToLower(os.Getenv("COLORTERM"))
	if colorterm == "truecolor" || colorterm == "24bit" {
		lipgloss.SetColorProfile(termenv.TrueColor)
	}

	cfg := loadConfig()
	applyDirConfig(cfg.IgnoreDirs, cfg.IncludeDirs)
	applyTheme(cfg.Theme)
	if cfg.CodeTheme != "" {
		setCodeTheme(cfg.CodeTheme)
	}
	root, err := initialModel(target, cfg)
	if err != nil {
		return err
	}

	program := tea.NewProgram(root, tea.WithAltScreen())
	_, err = program.Run()
	return err
}

// view determines which full-screen view the app is showing.
type view int

const (
	viewEditor view = iota
	viewTree
)

// app is the root Bubble Tea model. It owns the editor, the tree, and the
// picker overlay, and dispatches Update logic based on which view is active
// (and whether the picker is open on top of it).
type app struct {
	width  int
	height int

	// projectRoot is the directory marg considers "the project". It's the
	// cwd if launched with no args, the dir if launched with a dir arg, or
	// the parent of the file if launched with a file arg.
	projectRoot string

	cfg Config

	// superMode means marg was launched without arguments — `ctrl+p` should
	// reopen the super-mode index (machine-wide markdown) instead of the
	// project-scoped picker.
	superMode bool

	// watcher follows the currently open file on disk so external edits
	// (e.g. Claude Code rewriting the file in another tmux pane) trigger
	// an auto-reload.
	watcher *fileWatcher

	// zenMode hides the status bar and gives the editor the entire
	// terminal height — toggled with `:zen`.
	zenMode bool

	view    view
	editor  editor
	tree    tree
	picker  picker
	picking bool

	// chat is the AI conversation overlay. chatting is true while it's
	// shown; opening it is triggered by the editor's chatRequested flag.
	chat     chat
	chatting bool

	// statusMessage is a transient line shown in the status bar (e.g. "saved").
	statusMessage string

	// alternateFile is the path of the previously-open file, swapped on every
	// successful open. ctrl+6 / ctrl+^ jumps to it (vim's "alternate buffer").
	alternateFile string

	quitting bool
}

func initialModel(target startTarget, cfg Config) (app, error) {
	a := app{cfg: cfg}

	home, _ := os.UserHomeDir()

	switch target.kind {
	case targetDir:
		a.projectRoot = target.path
		a.tree = newTree(target.path)
		a.editor = newEditor("")
		a.view = viewTree
	case targetFile:
		a.projectRoot = parentDir(target.path)
		a.tree = newTree(a.projectRoot)
		ed, err := loadEditor(target.path)
		if err != nil {
			return a, err
		}
		a.editor = ed
		a.watcher = newFileWatcher(target.path)
		a.view = viewEditor
	case targetSuper:
		// No specific project; the editor sits idle until a file is picked.
		// The tree is built lazily on first :Ex — eagerly walking $HOME
		// would block the picker for many seconds.
		fallback := home
		if fallback == "" {
			fallback, _ = os.Getwd()
		}
		a.projectRoot = fallback
		a.tree = newTreeLazy(fallback)
		a.editor = newEditor("")
		a.view = viewEditor
	}

	a.editor.maxWidth = cfg.MaxWidth
	a.editor.codeMaxWidth = cfg.CodeMaxWidth
	a.editor.centerAbove = cfg.CenterAbove
	a.editor.ai = cfg.AI
	a.picker = newPicker()
	if target.kind == targetSuper {
		a.superMode = true
		a.picker.openSuper(cfg.SuperRoots)
		a.picking = true
	}
	return a, nil
}

func (a app) Init() tea.Cmd {
	var cmds []tea.Cmd
	if a.superMode {
		cmds = append(cmds, indexCmd(a.cfg.SuperRoots))
	}
	if a.watcher != nil {
		cmds = append(cmds, a.watcher.nextEventCmd())
	}
	return tea.Batch(cmds...)
}

// indexCmd kicks off a super-mode walk in the background. The result lands
// as an indexResultMsg.
func indexCmd(roots []string) tea.Cmd {
	return func() tea.Msg {
		return indexResultMsg{files: findMarkdownFiles(roots)}
	}
}

type indexResultMsg struct {
	files []string
}

func (a app) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch m := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = m.Width
		a.height = m.Height
		a.editor.resize(a.width, a.editorContentHeight())
		a.tree.resize(a.width, a.height-1)
		a.picker.resize(a.width, a.height)
		a.chat.resize(a.width, a.height)
		return a, nil

	case statusMsg:
		a.statusMessage = string(m)
		return a, nil

	case flashTickMsg:
		a.editor = a.editor.onFlashTick(m.gen)
		return a, nil

	case indexResultMsg:
		a.picker.setIndex(m.files)
		return a, nil

	case openFileMsg:
		ed, err := loadEditor(string(m))
		if err != nil {
			a.statusMessage = "open failed: " + err.Error()
			return a, nil
		}
		if a.editor.filepath != "" && a.editor.filepath != string(m) {
			a.alternateFile = a.editor.filepath
		}
		a.editor = ed
		a.editor.maxWidth = a.cfg.MaxWidth
		a.editor.codeMaxWidth = a.cfg.CodeMaxWidth
		a.editor.centerAbove = a.cfg.CenterAbove
		a.editor.ai = a.cfg.AI
		a.editor.resize(a.width, a.editorContentHeight())
		a.view = viewEditor
		a.picking = false
		// Swap in a watcher for the newly opened file.
		a.watcher.close()
		a.watcher = newFileWatcher(string(m))
		var cmd tea.Cmd
		if a.watcher != nil {
			cmd = a.watcher.nextEventCmd()
		}
		return a, cmd

	case fileChangedMsg:
		a.handleFileChanged()
		var cmd tea.Cmd
		if a.watcher != nil {
			cmd = a.watcher.nextEventCmd()
		}
		return a, cmd

	case zenToggleMsg:
		a.zenMode = !a.zenMode
		a.editor.resize(a.width, a.editorContentHeight())
		return a, nil

	case proofResultMsg:
		a.editor = a.editor.onProofResult(m)
		return a, nil

	case chatResultMsg:
		a.chat = a.chat.onResult(m)
		return a, nil

	case quitMsg:
		a.quitting = true
		return a, tea.Quit

	case tea.KeyMsg:
		// Chat overlay takes priority when open.
		if a.chatting {
			next, cmd := a.chat.update(m)
			a.chat = next
			if a.chat.cancelled {
				a.chatting = false
				a.chat.cancelled = false
				return a, nil
			}
			return a, cmd
		}

		// Picker takes priority when open.
		if a.picking {
			next, cmd := a.picker.update(m)
			a.picker = next
			if a.picker.cancelled {
				a.picking = false
				a.picker.cancelled = false
				return a, nil
			}
			if a.picker.chosen != "" {
				picked := a.picker.chosen
				a.picker.chosen = ""
				a.picking = false
				return a, openFileCmd(picked)
			}
			return a, cmd
		}

		// Global keys regardless of view.
		if handled, next, cmd := a.handleGlobalKey(m); handled {
			return next, cmd
		}

		switch a.view {
		case viewEditor:
			next, cmd := a.editor.update(m)
			a.editor = next
			if a.editor.openTreeRequested {
				a.editor.openTreeRequested = false
				a.view = viewTree
				a.tree.refresh()
				return a, nil
			}
			if a.editor.chatRequested {
				a.editor.chatRequested = false
				selection := a.editor.chatSelection
				a.editor.chatSelection = ""
				a.chat = newChat(a.cfg.AI, a.editor.buf.toString(), selection, a.editor.filepath)
				a.chat.resize(a.width, a.height)
				a.chatting = true
				return a, nil
			}
			if a.editor.quitRequested {
				return a, tea.Quit
			}
			return a, cmd
		case viewTree:
			next, cmd, openPath := a.tree.update(m)
			a.tree = next
			if openPath != "" {
				return a, openFileCmd(openPath)
			}
			if a.tree.backRequested {
				a.tree.backRequested = false
				if a.editor.filepath != "" {
					a.view = viewEditor
				} else {
					return a, tea.Quit
				}
			}
			return a, cmd
		}
	}
	return a, nil
}

func (a *app) handleGlobalKey(m tea.KeyMsg) (bool, tea.Model, tea.Cmd) {
	// Ctrl+P opens the fuzzy file picker from any view. Block while typing
	// in insert mode in the editor — that lets users type the literal char.
	if a.view == viewEditor && a.editor.mode == modeInsert {
		return false, *a, nil
	}
	switch m.String() {
	case "ctrl+p":
		var cmd tea.Cmd
		if a.superMode {
			a.picker.openSuper(a.cfg.SuperRoots)
			cmd = indexCmd(a.cfg.SuperRoots)
		} else {
			a.picker.open(a.projectRoot)
		}
		a.picker.resize(a.width, a.height)
		a.picking = true
		return true, *a, cmd
	case "ctrl+6", "ctrl+^":
		if a.alternateFile == "" {
			a.statusMessage = "no alternate file"
			return true, *a, nil
		}
		return true, *a, openFileCmd(a.alternateFile)
	}
	return false, *a, nil
}

func (a app) View() string {
	if a.quitting {
		return ""
	}

	var body string
	switch a.view {
	case viewEditor:
		body = a.editor.view()
	case viewTree:
		body = a.tree.view()
	}

	out := body
	if !a.zenMode {
		out = body + "\n" + a.renderStatusBar()
	}

	if a.picking {
		out = a.picker.overlay(out)
	}
	if a.chatting {
		out = a.chat.overlay(out)
	}
	return out
}

func (a app) editorContentHeight() int {
	if a.zenMode {
		if a.height <= 0 {
			return 1
		}
		return a.height
	}
	// Reserve one row for the status bar.
	if a.height <= 1 {
		return 1
	}
	return a.height - 1
}

// renderStatusBar builds the bottom status row. Content depends on view.
func (a app) renderStatusBar() string {
	if a.view == viewEditor {
		return a.editor.statusBar(a.width, a.statusMessage)
	}
	return a.tree.statusBar(a.width, a.statusMessage)
}

// handleFileChanged is called when the watcher reports the open file was
// touched on disk. If the buffer has unsaved changes, we don't clobber them
// — we just flash a warning. Otherwise we silently reload from disk so the
// user picks up edits that happened in another pane.
func (a *app) handleFileChanged() {
	path := a.editor.filepath
	if path == "" {
		return
	}
	data, err := os.ReadFile(path)
	if err != nil {
		// File might have been deleted or temporarily missing during an
		// atomic rename; just ignore.
		return
	}
	if string(data) == a.editor.buf.toString() {
		// Same content (likely our own save echoing back).
		return
	}
	if a.editor.dirty {
		a.editor.flash = "file changed on disk — :e! to discard local changes"
		return
	}
	a.editor.buf = bufferFromString(string(data))
	a.editor.clampCursor()
	a.editor.scrollToCursor()
	a.editor.flash = "reloaded from disk"
	a.editor.recordChange()
}

// --- messages ---

type statusMsg string
type openFileMsg string
type quitMsg struct{}
type zenToggleMsg struct{}

func openFileCmd(path string) tea.Cmd {
	return func() tea.Msg { return openFileMsg(path) }
}

// --- helpers ---

func parentDir(p string) string {
	d, err := os.Stat(p)
	if err != nil {
		return "."
	}
	if d.IsDir() {
		return p
	}
	// Trim trailing /file.md
	i := lastSep(p)
	if i < 0 {
		return "."
	}
	return p[:i]
}

func lastSep(p string) int {
	for i := len(p) - 1; i >= 0; i-- {
		if p[i] == os.PathSeparator || p[i] == '/' {
			return i
		}
	}
	return -1
}

