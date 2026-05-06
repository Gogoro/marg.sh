import SwiftUI
import AppKit

struct ContentView: View {
    @EnvironmentObject var appState: AppState

    var body: some View {
        NavigationSplitView {
            FileTreeView()
                .frame(minWidth: 220)
                .navigationSplitViewColumnWidth(min: 200, ideal: 260, max: 360)
        } detail: {
            VStack(spacing: 0) {
                MarkdownEditor()
                    .frame(maxWidth: .infinity, maxHeight: .infinity)
                StatusBar()
            }
        }
        .navigationTitle(currentTitle)
        .sheet(isPresented: $appState.showingPicker) {
            FuzzyPicker()
                .frame(minWidth: 540, minHeight: 360)
        }
    }

    private var currentTitle: String {
        if let url = appState.currentFileURL {
            return url.lastPathComponent + (appState.isDirty ? " ●" : "")
        }
        return "marg"
    }
}

struct StatusBar: View {
    @EnvironmentObject var appState: AppState

    var body: some View {
        HStack(spacing: 12) {
            Text(modeLabel)
                .font(.system(.caption, design: .monospaced).weight(.semibold))
                .foregroundColor(.white)
                .padding(.horizontal, 6)
                .padding(.vertical, 2)
                .background(modeColor)
                .cornerRadius(3)

            if let url = appState.currentFileURL {
                Text(url.path.replacingOccurrences(of: NSHomeDirectory(), with: "~"))
                    .foregroundColor(Color(NSColor.secondaryLabelColor))
                    .lineLimit(1)
                    .truncationMode(.middle)
            } else {
                Text("no file open")
                    .foregroundColor(Color(NSColor.tertiaryLabelColor))
            }

            if appState.isDirty {
                Text("●")
                    .foregroundColor(Color(NSColor.systemOrange))
            }

            Spacer()

            if appState.vimMode == .commandLine {
                Text(":\(appState.commandLineBuffer)")
                    .font(.system(.body, design: .monospaced))
                    .foregroundColor(Color(NSColor.labelColor))
            } else if let message = appState.statusMessage {
                Text(message)
                    .foregroundColor(Color(NSColor.secondaryLabelColor))
                    .lineLimit(1)
            }
        }
        .padding(.horizontal, 12)
        .padding(.vertical, 6)
        .background(Color(NSColor.windowBackgroundColor))
        .overlay(Divider(), alignment: .top)
        .font(.caption)
    }

    private var modeLabel: String {
        switch appState.vimMode {
        case .normal: return "NORMAL"
        case .insert: return "INSERT"
        case .visual: return "VISUAL"
        case .commandLine: return "CMD"
        }
    }

    private var modeColor: Color {
        switch appState.vimMode {
        case .normal: return Color(NSColor.systemBlue)
        case .insert: return Color(NSColor.systemGreen)
        case .visual: return Color(NSColor.systemPurple)
        case .commandLine: return Color(NSColor.systemOrange)
        }
    }
}
