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
        // hidden noise we explicitly skip even though dot-dirs are allowed by default
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
        var flat: [URL] = []
        let nodes = walkDirectory(rootURL, depth: 0, flatList: &flat)
        return FileTreeWalkResult(tree: nodes, flatFiles: flat)
    }

    private func walkDirectory(_ url: URL, depth: Int, flatList: inout [URL]) -> [FileNode] {
        let manager = FileManager.default
        guard let entries = try? manager.contentsOfDirectory(
            at: url,
            includingPropertiesForKeys: [.isDirectoryKey],
            options: []
        ) else {
            return []
        }

        var directories: [URL] = []
        var files: [URL] = []
        for entry in entries {
            let name = entry.lastPathComponent
            let isDirectory = (try? entry.resourceValues(forKeys: [.isDirectoryKey]).isDirectory) ?? false
            if isDirectory {
                if ignoredDirectoryNames.contains(name) { continue }
                if userIgnored.contains(name) { continue }
                directories.append(entry)
            } else {
                if name.hasPrefix(".") { continue }
                if markdownExtensions.contains(entry.pathExtension.lowercased()) {
                    files.append(entry)
                }
            }
        }

        directories.sort { $0.lastPathComponent.localizedStandardCompare($1.lastPathComponent) == .orderedAscending }
        files.sort { $0.lastPathComponent.localizedStandardCompare($1.lastPathComponent) == .orderedAscending }

        var nodes: [FileNode] = []
        for directory in directories {
            let children = walkDirectory(directory, depth: depth + 1, flatList: &flatList)
            if !children.isEmpty {
                nodes.append(FileNode(
                    url: directory,
                    name: directory.lastPathComponent,
                    isDirectory: true,
                    children: children
                ))
            }
        }
        for file in files {
            flatList.append(file)
            nodes.append(FileNode(
                url: file,
                name: file.lastPathComponent,
                isDirectory: false,
                children: nil
            ))
        }
        return nodes
    }
}
