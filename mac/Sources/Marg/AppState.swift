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
    @Published var vimEnabled: Bool = false
    @Published var statusMessage: String?
    @Published var commandLineBuffer: String = ""
    @Published var quitRequested: Bool = false
    @Published var userIgnoredFolders: [String]
    @Published var showingIgnoredManager: Bool = false
    @Published var isIndexing: Bool = false

    private let userIgnoredKey = "userIgnoredFolders"
    private let indexQueue = DispatchQueue(label: "marg.index", qos: .userInitiated)
    private var indexGeneration: Int = 0
    private var watcher: FileWatcher?
    private var statusClearTimer: Timer?

    init() {
        self.rootURL = FileManager.default.homeDirectoryForCurrentUser
        self.userIgnoredFolders = UserDefaults.standard.stringArray(forKey: "userIgnoredFolders") ?? []

        let warmStartWalker = FileTreeWalker(rootURL: rootURL, userIgnored: Set(userIgnoredFolders))
        if let cached = warmStartWalker.loadCachedResult() {
            self.fileTree = cached.tree
            self.allMarkdownFiles = cached.flatFiles
        }
        refreshIndex()
    }

    func refreshIndex() {
        indexGeneration += 1
        let generation = indexGeneration
        let root = rootURL
        let ignored = Set(userIgnoredFolders)
        isIndexing = true

        indexQueue.async { [weak self] in
            let walker = FileTreeWalker(rootURL: root, userIgnored: ignored)
            let result = walker.walk()
            DispatchQueue.main.async {
                guard let self = self, self.indexGeneration == generation else { return }
                self.fileTree = result.tree
                self.allMarkdownFiles = result.flatFiles
                self.isIndexing = false
            }
        }
    }

    func addIgnoredFolder(name: String) {
        let trimmed = name.trimmingCharacters(in: .whitespaces)
        guard !trimmed.isEmpty, !userIgnoredFolders.contains(trimmed) else { return }
        userIgnoredFolders.append(trimmed)
        userIgnoredFolders.sort()
        persistIgnoreList()
        fileTree = filterTreeRemoving(basename: trimmed, from: fileTree)
        allMarkdownFiles = allMarkdownFiles.filter { url in
            !url.pathComponents.contains(trimmed)
        }
        flashStatus("ignoring '\(trimmed)'")
        refreshIndex()
    }

    func removeIgnoredFolder(name: String) {
        userIgnoredFolders.removeAll { $0 == name }
        persistIgnoreList()
        flashStatus("unignored '\(name)'")
        refreshIndex()
    }

    private func filterTreeRemoving(basename: String, from nodes: [FileNode]) -> [FileNode] {
        var result: [FileNode] = []
        for node in nodes {
            if node.isDirectory && node.name == basename {
                continue
            }
            if let children = node.children {
                let filteredChildren = filterTreeRemoving(basename: basename, from: children)
                if node.isDirectory && filteredChildren.isEmpty {
                    continue
                }
                result.append(FileNode(
                    url: node.url,
                    name: node.name,
                    isDirectory: node.isDirectory,
                    children: filteredChildren
                ))
            } else {
                result.append(node)
            }
        }
        return result
    }

    private func persistIgnoreList() {
        UserDefaults.standard.set(userIgnoredFolders, forKey: userIgnoredKey)
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
