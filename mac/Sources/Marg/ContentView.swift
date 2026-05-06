import SwiftUI
import AppKit

struct ContentView: View {
    @EnvironmentObject var appState: AppState

    var body: some View {
        ZStack(alignment: .top) {
            HStack(spacing: 0) {
                FileTreeView()
                    .frame(width: Theme.sidebarWidth)
                    .background(Theme.sidebarBackgroundColor)

                Rectangle()
                    .fill(Theme.dividerColor)
                    .frame(width: 1)

                VStack(spacing: 0) {
                    MarkdownEditor()
                        .frame(maxWidth: .infinity, maxHeight: .infinity)
                    StatusBar()
                }
                .background(Theme.editorBackgroundColor)
            }
            .background(Theme.editorBackgroundColor)

            if appState.showingPicker {
                FuzzyPickerOverlay()
            }
        }
        .preferredColorScheme(.light)
        .sheet(isPresented: $appState.showingIgnoredManager) {
            IgnoredFoldersView()
                .environmentObject(appState)
        }
    }
}

struct StatusBar: View {
    @EnvironmentObject var appState: AppState

    var body: some View {
        HStack(spacing: 12) {
            if appState.vimEnabled && appState.vimMode == .commandLine {
                Text(":\(appState.commandLineBuffer)")
                    .font(.system(size: 13, design: .monospaced))
                    .foregroundColor(Theme.primaryTextColor)
            } else {
                Text(leftLabel)
                    .font(Theme.statusFont)
                    .foregroundColor(Theme.mutedTextColor)
                    .lineLimit(1)
                    .truncationMode(.middle)
            }

            Spacer()

            if appState.isDirty {
                Circle()
                    .fill(Theme.dirtyDotColor)
                    .frame(width: 6, height: 6)
            }

            if let message = appState.statusMessage {
                Text(message)
                    .font(Theme.statusFont)
                    .foregroundColor(Theme.mutedTextColor)
                    .lineLimit(1)
            }

            if appState.vimEnabled {
                Text(modeLabel)
                    .font(.system(size: 10, weight: .medium).smallCaps())
                    .foregroundColor(Theme.mutedTextColor)
                    .tracking(0.4)
            }
        }
        .padding(.horizontal, 18)
        .frame(height: Theme.statusBarHeight)
        .background(
            Rectangle()
                .fill(Theme.editorBackgroundColor)
                .overlay(Rectangle().fill(Theme.dividerColor).frame(height: 1), alignment: .top)
        )
    }

    private var leftLabel: String {
        if let url = appState.currentFileURL {
            return url.path.replacingOccurrences(of: NSHomeDirectory(), with: "~")
        }
        return "no file open"
    }

    private var modeLabel: String {
        switch appState.vimMode {
        case .normal: return "normal"
        case .insert: return "insert"
        case .visual: return "visual"
        case .commandLine: return "command"
        }
    }
}
