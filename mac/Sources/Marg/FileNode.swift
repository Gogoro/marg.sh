import Foundation

struct FileNode: Identifiable, Hashable {
    let id: URL
    let url: URL
    let name: String
    let isDirectory: Bool
    let children: [FileNode]?

    init(url: URL, name: String, isDirectory: Bool, children: [FileNode]?) {
        self.id = url
        self.url = url
        self.name = name
        self.isDirectory = isDirectory
        self.children = children
    }
}
