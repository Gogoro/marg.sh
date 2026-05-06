import Combine
import Foundation
import SwiftUI

final class AppState: ObservableObject {
    @Published var rootURL: URL
    @Published var currentFileURL: URL?
    @Published var text: String = ""
    @Published var isDirty: Bool = false
    @Published var fileTree: [FileNode] = []
    @Published var allMarkdownFiles: [URL] = []
    @Published var showingPicker: Bool = false
    @Published var vimMode: VimMode = .normal
    @Published var statusMessage: String?
    @Published var commandLineBuffer: String = ""
    @Published var quitRequested: Bool = false

    private var watcher: FileWatcher?
    private var statusClearTimer: Timer?

    init() {
        self.rootURL = FileManager.default.homeDirectoryForCurrentUser
        refreshIndex()
    }

    func refreshIndex() {
        let walker = FileTreeWalker(rootURL: rootURL)
        let result = walker.walk()
        self.fileTree = result.tree
        self.allMarkdownFiles = result.flatFiles
    }

    func loadFile(_ url: URL) {
        do {
            let content = try String(contentsOf: url, encoding: .utf8)
            self.text = content
            self.currentFileURL = url
            self.isDirty = false
            self.vimMode = .normal
            startWatching(url)
            flashStatus("opened \(url.lastPathComponent)")
        } catch {
            flashStatus("open failed: \(error.localizedDescription)")
        }
    }

    func saveCurrentFile() {
        guard let url = currentFileURL else { return }
        do {
            try text.write(to: url, atomically: true, encoding: .utf8)
            self.isDirty = false
            flashStatus("saved")
        } catch {
            flashStatus("save failed: \(error.localizedDescription)")
        }
    }

    func discardAndReload() {
        guard let url = currentFileURL else { return }
        if let content = try? String(contentsOf: url, encoding: .utf8) {
            self.text = content
            self.isDirty = false
            flashStatus("reloaded from disk")
        }
    }

    func markDirty() {
        if !isDirty { isDirty = true }
    }

    func flashStatus(_ message: String) {
        self.statusMessage = message
        statusClearTimer?.invalidate()
        statusClearTimer = Timer.scheduledTimer(withTimeInterval: 3, repeats: false) { [weak self] _ in
            DispatchQueue.main.async {
                self?.statusMessage = nil
            }
        }
    }

    func runCommandLine(_ command: String) {
        let trimmed = command.trimmingCharacters(in: .whitespaces)
        switch trimmed {
        case "w":
            saveCurrentFile()
        case "q":
            if isDirty {
                flashStatus("unsaved changes — :q! to force or :wq to save")
            } else {
                quitRequested = true
                NSApplication.shared.terminate(nil)
            }
        case "q!":
            NSApplication.shared.terminate(nil)
        case "wq", "x":
            saveCurrentFile()
            NSApplication.shared.terminate(nil)
        case "e!":
            discardAndReload()
        default:
            flashStatus("unknown command: :\(trimmed)")
        }
        self.commandLineBuffer = ""
        self.vimMode = .normal
    }

    private func startWatching(_ url: URL) {
        watcher?.stop()
        watcher = FileWatcher(url: url) { [weak self] in
            DispatchQueue.main.async {
                self?.handleExternalChange()
            }
        }
        watcher?.start()
    }

    private func handleExternalChange() {
        guard let url = currentFileURL else { return }
        if isDirty {
            flashStatus("file changed on disk — :e! to discard")
            return
        }
        if let content = try? String(contentsOf: url, encoding: .utf8), content != text {
            self.text = content
            flashStatus("reloaded from disk")
        }
    }
}
