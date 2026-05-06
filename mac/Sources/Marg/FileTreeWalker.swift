import Foundation

struct FileTreeWalkResult {
    let tree: [FileNode]
    let flatFiles: [URL]
}

final class FileTreeWalker {
    let rootURL: URL

    private let ignoredDirectoryNames: Set<String> = [
        "node_modules", "vendor", "Pods", "Carthage",
        "target", "build", "dist", "DerivedData", "coverage",
        "Library", "Applications", "Pictures", "Movies", "Music",
        "Public"
    ]

    private let markdownExtensions: Set<String> = ["md", "markdown"]

    init(rootURL: URL) {
        self.rootURL = rootURL
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
            options: [.skipsHiddenFiles]
        ) else {
            return []
        }

        var directories: [URL] = []
        var files: [URL] = []
        for entry in entries {
            let isDirectory = (try? entry.resourceValues(forKeys: [.isDirectoryKey]).isDirectory) ?? false
            if isDirectory {
                if !ignoredDirectoryNames.contains(entry.lastPathComponent) {
                    directories.append(entry)
                }
            } else if markdownExtensions.contains(entry.pathExtension.lowercased()) {
                files.append(entry)
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
