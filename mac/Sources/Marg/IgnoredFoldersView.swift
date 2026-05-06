import SwiftUI
import AppKit

struct IgnoredFoldersView: View {
    @EnvironmentObject var appState: AppState

    var body: some View {
        VStack(spacing: 0) {
            header
            Rectangle().fill(Theme.dividerColor).frame(height: 1)

            if appState.userIgnoredFolders.isEmpty {
                emptyState
            } else {
                listView
            }
        }
        .frame(width: 460, height: 380)
        .background(Theme.editorBackgroundColor)
    }

    private var header: some View {
        HStack {
            Text("Ignored folders")
                .font(.system(size: 14, weight: .semibold))
                .foregroundColor(Theme.primaryTextColor)
            Spacer()
            Button("Done") {
                appState.showingIgnoredManager = false
            }
            .keyboardShortcut(.defaultAction)
        }
        .padding(.horizontal, 18)
        .padding(.vertical, 14)
    }

    private var emptyState: some View {
        VStack(spacing: 8) {
            Spacer()
            Text("Nothing ignored yet")
                .font(.system(size: 13))
                .foregroundColor(Theme.secondaryTextColor)
            Text("Right-click a folder in the sidebar to ignore it.")
                .font(.system(size: 11))
                .foregroundColor(Theme.mutedTextColor)
            Spacer()
        }
        .frame(maxWidth: .infinity, maxHeight: .infinity)
    }

    private var listView: some View {
        ScrollView {
            LazyVStack(spacing: 0) {
                ForEach(appState.userIgnoredFolders, id: \.self) { name in
                    IgnoredFolderRow(name: name)
                    Rectangle()
                        .fill(Theme.dividerColor)
                        .frame(height: 1)
                }
            }
        }
    }
}

private struct IgnoredFolderRow: View {
    @EnvironmentObject var appState: AppState
    let name: String

    @State private var hovered: Bool = false

    var body: some View {
        HStack(spacing: 10) {
            Image(systemName: "folder")
                .font(.system(size: 12))
                .foregroundColor(Theme.mutedTextColor)
                .frame(width: 14)

            Text(name)
                .font(.system(size: 13))
                .foregroundColor(Theme.primaryTextColor)

            Spacer()

            Button(action: {
                appState.removeIgnoredFolder(name: name)
            }) {
                Text("Remove")
                    .font(.system(size: 12))
                    .foregroundColor(hovered ? Color(NSColor.systemRed) : Theme.secondaryTextColor)
            }
            .buttonStyle(.plain)
        }
        .padding(.horizontal, 18)
        .padding(.vertical, 10)
        .contentShape(Rectangle())
        .onHover { hovered = $0 }
    }
}
