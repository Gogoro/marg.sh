import SwiftUI
import AppKit

struct FileTreeView: View {
    @EnvironmentObject var appState: AppState

    var body: some View {
        VStack(spacing: 0) {
            Color.clear.frame(height: Theme.titleBarInset)

            HStack {
                Text("FILES")
                    .font(Theme.sidebarHeaderFont)
                    .foregroundColor(Theme.mutedTextColor)
                    .tracking(0.6)
                Spacer()
                Button(action: { appState.showingPicker = true }) {
                    Image(systemName: "magnifyingglass")
                        .font(.system(size: 11, weight: .medium))
                        .foregroundColor(Theme.mutedTextColor)
                }
                .buttonStyle(.plain)
                .help("Find file (⌘P)")
            }
            .padding(.horizontal, 16)
            .padding(.bottom, 10)

            ScrollViewReader { proxy in
                ScrollView {
                    if appState.isIndexing && appState.fileTree.isEmpty {
                        IndexingPlaceholder(rootName: appState.rootURL.lastPathComponent)
                    } else {
                        LazyVStack(alignment: .leading, spacing: 1) {
                            ForEach(appState.fileTree) { node in
                                FileTreeNodeView(node: node, depth: 0)
                            }
                        }
                        .padding(.horizontal, 8)
                        .padding(.bottom, 12)
                    }
                }
                .task(id: appState.currentFileURL) {
                    guard let url = appState.currentFileURL else { return }
                    try? await Task.sleep(nanoseconds: 120_000_000)
                    withAnimation(.easeOut(duration: 0.2)) {
                        proxy.scrollTo(url, anchor: .center)
                    }
                }
            }
        }
    }
}

private struct IndexingPlaceholder: View {
    let rootName: String

    var body: some View {
        VStack(spacing: 10) {
            ProgressView()
                .controlSize(.small)
            Text("scanning \(rootName)…")
                .font(Theme.sidebarFont)
                .foregroundColor(Theme.mutedTextColor)
        }
        .frame(maxWidth: .infinity)
        .padding(.top, 24)
        .padding(.horizontal, 16)
    }
}

private struct FileTreeNodeView: View {
    @EnvironmentObject var appState: AppState
    let node: FileNode
    let depth: Int

    @State private var hovered: Bool = false

    private var isExpanded: Bool {
        appState.expandedDirectories.contains(node.url)
    }

    var body: some View {
        VStack(alignment: .leading, spacing: 1) {
            row
            if node.isDirectory && isExpanded, let children = node.children {
                ForEach(children) { child in
                    FileTreeNodeView(node: child, depth: depth + 1)
                }
            }
        }
    }

    private var row: some View {
        HStack(spacing: 4) {
            Color.clear.frame(width: CGFloat(depth) * 14)

            if node.isDirectory {
                Image(systemName: isExpanded ? "chevron.down" : "chevron.right")
                    .font(.system(size: 9, weight: .medium))
                    .foregroundColor(Theme.mutedTextColor)
                    .frame(width: 12)
            } else {
                Color.clear.frame(width: 12)
            }

            Text(node.name)
                .font(Theme.sidebarFont)
                .foregroundColor(textColor)
                .lineLimit(1)
                .truncationMode(.middle)

            Spacer(minLength: 0)
        }
        .padding(.horizontal, 8)
        .padding(.vertical, 4)
        .background(rowBackground)
        .clipShape(RoundedRectangle(cornerRadius: 4))
        .contentShape(Rectangle())
        .onHover { hovered = $0 }
        .onTapGesture {
            if node.isDirectory {
                appState.toggleDirectoryExpansion(node.url)
            } else {
                appState.loadFile(node.url)
            }
        }
        .contextMenu {
            Button("Reveal in Finder") {
                NSWorkspace.shared.activateFileViewerSelecting([node.url])
            }
            if node.isDirectory {
                Divider()
                Button("Add '\(node.name)' to ignore list") {
                    appState.addIgnoredFolder(name: node.name)
                }
                Divider()
                Button("Manage Ignored Folders…") {
                    appState.showingIgnoredManager = true
                }
            }
        }
        .id(node.url)
    }

    private var textColor: Color {
        if isCurrent { return Theme.primaryTextColor }
        if node.isDirectory { return Theme.secondaryTextColor }
        return Theme.primaryTextColor
    }

    private var rowBackground: Color {
        if isCurrent { return Theme.sidebarSelectionColor }
        if hovered { return Theme.sidebarHoverColor }
        return Color.clear
    }

    private var isCurrent: Bool {
        appState.currentFileURL == node.url
    }
}
