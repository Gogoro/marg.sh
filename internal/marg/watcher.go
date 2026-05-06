package marg

import (
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/fsnotify/fsnotify"
)

// fileWatcher watches the directory containing the currently open file. It
// emits a fileChangedMsg whenever the open file is written, created, or
// renamed externally (typical when Claude Code or another tool edits the
// file behind marg's back).
//
// We watch the parent directory rather than the file directly because many
// editors atomic-rename: they write to a temp file and rename it onto the
// target, which detaches a direct watch from the new inode.
type fileWatcher struct {
	w      *fsnotify.Watcher
	target string // absolute path of the file we care about
	events chan fileChangedMsg
	done   chan struct{}
}

// fileChangedMsg fires when the watcher sees the open file modified on disk.
type fileChangedMsg struct{}

func newFileWatcher(path string) *fileWatcher {
	if path == "" {
		return nil
	}
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return nil
	}
	dir := filepath.Dir(path)
	if err := w.Add(dir); err != nil {
		w.Close()
		return nil
	}

	fw := &fileWatcher{
		w:      w,
		target: filepath.Clean(path),
		events: make(chan fileChangedMsg, 8),
		done:   make(chan struct{}),
	}
	go fw.loop()
	return fw
}

func (fw *fileWatcher) loop() {
	for {
		select {
		case <-fw.done:
			return
		case ev, ok := <-fw.w.Events:
			if !ok {
				return
			}
			if filepath.Clean(ev.Name) != fw.target {
				continue
			}
			if ev.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Rename) == 0 {
				continue
			}
			// Coalesce: drop the event if the channel is already full.
			select {
			case fw.events <- fileChangedMsg{}:
			default:
			}
		case _, ok := <-fw.w.Errors:
			if !ok {
				return
			}
		}
	}
}

func (fw *fileWatcher) close() {
	if fw == nil {
		return
	}
	close(fw.done)
	fw.w.Close()
}

// nextEventCmd returns a tea.Cmd that blocks until the watcher fires once.
// After handling the message the caller should schedule another call to keep
// listening.
func (fw *fileWatcher) nextEventCmd() tea.Cmd {
	if fw == nil {
		return nil
	}
	return func() tea.Msg {
		select {
		case <-fw.done:
			return nil
		case msg, ok := <-fw.events:
			if !ok {
				return nil
			}
			return msg
		}
	}
}
