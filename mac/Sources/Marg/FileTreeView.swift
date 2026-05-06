import SwiftUI
import AppKit

struct FileTreeView: View {
    @EnvironmentObject var appState: AppState

    var body: some View {
        List(appState.fileTree, children: \.children) { node in
            FileTreeRow(node: node)
                .contentShape(Rectangle())
                .onTapGesture {
                    if !node.isDirectory {
                        appState.loadFile(node.url)
                    }
                }
        }
        .listStyle(.sidebar)
    }
}

private struct FileTreeRow: View {
    @EnvironmentObject var appState: AppState
    let node: FileNode

    var body: some View {
        HStack(spacing: 6) {
            Image(systemName: node.isDirectory ? "folder" : "doc.text")
                .foregroundColor(.secondary)
                .frame(width: 14)
            Text(node.name)
                .lineLimit(1)
                .truncationMode(.middle)
                .foregroundColor(isCurrent ? Color(NSColor.controlAccentColor) : Color(NSColor.labelColor))
        }
    }

    private var isCurrent: Bool {
        appState.currentFileURL == node.url
    }
}
