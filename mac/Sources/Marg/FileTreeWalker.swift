import Foundation

struct FileTreeWalkResult {
    let tree: [FileNode]
    let flatFiles: [URL]
}

final class FileTreeWalker {
    let rootURL: URL
    private let userIgnored: Set<String>

    private let ignoredDirectoryNames: Set<String> = [
        "node_modules", "vendor", "Pods", "Carthage",
        "target", "build", "dist", "DerivedData", "coverage",
        "Library", "Applications", "Pictures", "Movies", "Music",
        "Public",
        ".git", ".svn", ".hg", ".cache", ".npm", ".yarn", ".pnpm",
        ".gradle", ".nuget", ".dotnet", ".terraform",
        ".next", ".nuxt", ".turbo", ".swiftpm", ".build",
        ".Trash", ".Trashes", ".idea", ".vscode-server",
        ".rustup", ".cargo", ".rbenv", ".pyenv", ".nvm", ".fnm",
        ".local", ".bun", ".deno", ".docker"
    ]

    private let markdownExtensions: Set<String> = ["md", "markdown"]

    init(rootURL: URL, userIgnored: Set<String> = []) {
        self.rootURL = rootURL
        self.userIgnored = userIgnored
    }

    func walk() -> FileTreeWalkResult {
        let files = enumerateMarkdownFiles()
        let tree = buildTree(from: files)
        let sortedFlat = files.sorted {
            $0.path.localizedStandardCompare($1.path) == .orderedAscending
        }
        saveCachedFileList(sortedFlat)
        return FileTreeWalkResult(tree: tree, flatFiles: sortedFlat)
    }

    func loadCachedResult() -> FileTreeWalkResult? {
        guard let cacheFileURL = Self.cacheFileURL() else { return nil }
        guard let data = try? Data(contentsOf: cacheFileURL) else { return nil }
        guard let paths = try? JSONDecoder().decode([String].self, from: data) else { return nil }
        let files = paths.map { URL(fileURLWithPath: $0) }
        let tree = buildTree(from: files)
        return FileTreeWalkResult(tree: tree, flatFiles: files)
    }

    private func saveCachedFileList(_ files: [URL]) {
        guard let cacheFileURL = Self.cacheFileURL() else { return }
        let paths = files.map { $0.path }
        guard let data = try? JSONEncoder().encode(paths) else { return }
        try? data.write(to: cacheFileURL, options: .atomic)
    }

    private static func cacheFileURL() -> URL? {
        guard let cachesDirectory = FileManager.default
            .urls(for: .cachesDirectory, in: .userDomainMask)
            .first else { return nil }
        let margCacheDirectory = cachesDirectory.appendingPathComponent("marg", isDirectory: true)
        try? FileManager.default.createDirectory(at: margCacheDirectory, withIntermediateDirectories: true)
        return margCacheDirectory.appendingPathComponent("index.json")
    }

    private func enumerateMarkdownFiles() -> [URL] {
        if let fd = locateBinary("fd"), let result = runFd(at: fd) {
            return result
        }
        if let rg = locateBinary("rg"), let result = runRipgrep(at: rg) {
            return result
        }
        return fallbackWalk()
    }

    private func locateBinary(_ name: String) -> URL? {
        let candidates = [
            "/opt/homebrew/bin/\(name)",
            "/usr/local/bin/\(name)",
            "/usr/bin/\(name)",
            "/opt/local/bin/\(name)"
        ]
        for path in candidates {
            if FileManager.default.isExecutableFile(atPath: path) {
                return URL(fileURLWithPath: path)
            }
        }
        return nil
    }

    private func runFd(at binary: URL) -> [URL]? {
        var arguments: [String] = [
            "--type", "f",
            "--extension", "md",
            "--extension", "markdown",
            "--hidden",
            "--absolute-path"
        ]
        for name in ignoredDirectoryNames {
            arguments.append("--exclude")
            arguments.append(name)
        }
        for name in userIgnored {
            arguments.append("--exclude")
            arguments.append(name)
        }
        arguments.append(".")
        arguments.append(rootURL.path)
        return runEnumeratorProcess(binary: binary, arguments: arguments)
    }

    private func runRipgrep(at binary: URL) -> [URL]? {
        var arguments: [String] = [
            "--files",
            "--hidden",
            "--type", "md"
        ]
        for name in ignoredDirectoryNames {
            arguments.append("--glob")
            arguments.append("!\(name)")
        }
        for name in userIgnored {
            arguments.append("--glob")
            arguments.append("!\(name)")
        }
        arguments.append(rootURL.path)
        return runEnumeratorProcess(binary: binary, arguments: arguments)
    }

    private func runEnumeratorProcess(binary: URL, arguments: [String]) -> [URL]? {
        let process = Process()
        process.executableURL = binary
        process.arguments = arguments

        let outputPipe = Pipe()
        process.standardOutput = outputPipe
        process.standardError = Pipe()

        do {
            try process.run()
        } catch {
            return nil
        }
        process.waitUntilExit()

        // fd exits 0 always. rg exits 0 with matches, 1 with no matches. Both fine.
        guard process.terminationStatus == 0 || process.terminationStatus == 1 else {
            return nil
        }

        let data = outputPipe.fileHandleForReading.readDataToEndOfFile()
        guard let output = String(data: data, encoding: .utf8) else { return nil }

        return output
            .split(separator: "\n", omittingEmptySubsequences: true)
            .map { URL(fileURLWithPath: String($0)) }
    }

    private func fallbackWalk() -> [URL] {
        var flat: [URL] = []
        fallbackWalkDirectory(rootURL, into: &flat)
        return flat
    }

    private func fallbackWalkDirectory(_ url: URL, into list: inout [URL]) {
        let manager = FileManager.default
        guard let entries = try? manager.contentsOfDirectory(
            at: url,
            includingPropertiesForKeys: [.isDirectoryKey],
            options: []
        ) else { return }

        for entry in entries {
            let name = entry.lastPathComponent
            let isDirectory = (try? entry.resourceValues(forKeys: [.isDirectoryKey]).isDirectory) ?? false
            if isDirectory {
                if ignoredDirectoryNames.contains(name) { continue }
                if userIgnored.contains(name) { continue }
                fallbackWalkDirectory(entry, into: &list)
            } else {
                if name.hasPrefix(".") { continue }
                if markdownExtensions.contains(entry.pathExtension.lowercased()) {
                    list.append(entry)
                }
            }
        }
    }

    private func buildTree(from files: [URL]) -> [FileNode] {
        let rootPath = rootURL.path
        let prefix = rootPath.hasSuffix("/") ? rootPath : rootPath + "/"
        let rootBuilder = TreeBuilder(url: rootURL, name: rootURL.lastPathComponent, isDirectory: true)

        for file in files {
            let path = file.path
            guard path.hasPrefix(prefix) else { continue }
            let relative = String(path.dropFirst(prefix.count))
            let components = relative
                .split(separator: "/", omittingEmptySubsequences: true)
                .map(String.init)
            insertFile(file, components: components, into: rootBuilder, currentURL: rootURL)
        }

        return rootBuilder.toFileNode().children ?? []
    }

    private func insertFile(_ file: URL, components: [String], into parent: TreeBuilder, currentURL: URL) {
        guard let head = components.first else { return }
        let rest = Array(components.dropFirst())

        if rest.isEmpty {
            parent.children[head] = TreeBuilder(url: file, name: head, isDirectory: false)
            return
        }

        let directoryURL = currentURL.appendingPathComponent(head)
        let child: TreeBuilder
        if let existing = parent.children[head] {
            child = existing
        } else {
            child = TreeBuilder(url: directoryURL, name: head, isDirectory: true)
            parent.children[head] = child
        }
        insertFile(file, components: rest, into: child, currentURL: directoryURL)
    }
}

private final class TreeBuilder {
    let url: URL
    let name: String
    let isDirectory: Bool
    var children: [String: TreeBuilder] = [:]

    init(url: URL, name: String, isDirectory: Bool) {
        self.url = url
        self.name = name
        self.isDirectory = isDirectory
    }

    func toFileNode() -> FileNode {
        if !isDirectory {
            return FileNode(url: url, name: name, isDirectory: false, children: nil)
        }
        let sorted = children.values
            .map { $0.toFileNode() }
            .sorted { a, b in
                if a.isDirectory != b.isDirectory { return a.isDirectory }
                return a.name.localizedStandardCompare(b.name) == .orderedAscending
            }
        return FileNode(url: url, name: name, isDirectory: true, children: sorted)
    }
}
