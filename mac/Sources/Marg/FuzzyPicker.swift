import SwiftUI
import AppKit

struct FuzzyPicker: View {
    @EnvironmentObject var appState: AppState
    @Environment(\.dismiss) private var dismiss

    @State private var query: String = ""
    @State private var cursor: Int = 0
    @FocusState private var inputFocused: Bool

    var body: some View {
        VStack(spacing: 0) {
            HStack(spacing: 8) {
                Text("›")
                    .font(.system(.title3, design: .monospaced))
                    .foregroundColor(.secondary)
                TextField("type to filter", text: $query)
                    .textFieldStyle(.plain)
                    .font(.system(.title3))
                    .focused($inputFocused)
                    .onSubmit { openSelection() }
                    .onChange(of: query) { _, _ in cursor = 0 }
                Text("\(matches.count)")
                    .font(.caption)
                    .foregroundColor(.secondary)
            }
            .padding(.horizontal, 14)
            .padding(.vertical, 12)

            Divider()

            ScrollViewReader { proxy in
                List {
                    ForEach(Array(matches.enumerated()), id: \.element.url) { index, item in
                        FuzzyPickerRow(displayPath: item.display, isSelected: index == cursor)
                            .id(item.url)
                            .onTapGesture { openItem(item.url) }
                    }
                }
                .listStyle(.plain)
                .onChange(of: cursor) { _, newValue in
                    if newValue < matches.count {
                        proxy.scrollTo(matches[newValue].url, anchor: .center)
                    }
                }
            }
        }
        .background(KeyCatcher(
            onMoveUp: { moveCursor(-1) },
            onMoveDown: { moveCursor(1) },
            onCancel: { close() },
            onSubmit: { openSelection() }
        ))
        .onAppear {
            inputFocused = true
        }
    }

    private var matches: [(url: URL, display: String)] {
        let candidates = appState.allMarkdownFiles
        let displayed = candidates.map { displayPath(for: $0) }
        let scored = FuzzyMatcher.match(query: query, candidates: displayed)
        return scored.map { (candidates[$0.index], displayed[$0.index]) }
    }

    private func displayPath(for url: URL) -> String {
        let home = NSHomeDirectory()
        let path = url.path
        if path.hasPrefix(home + "/") {
            return "~" + String(path.dropFirst(home.count))
        }
        return path
    }

    private func moveCursor(_ delta: Int) {
        let maxIndex = max(0, matches.count - 1)
        cursor = min(maxIndex, max(0, cursor + delta))
    }

    private func openSelection() {
        guard cursor < matches.count else { return }
        openItem(matches[cursor].url)
    }

    private func openItem(_ url: URL) {
        appState.loadFile(url)
        close()
    }

    private func close() {
        appState.showingPicker = false
        dismiss()
    }
}

private struct FuzzyPickerRow: View {
    let displayPath: String
    let isSelected: Bool

    var body: some View {
        HStack {
            Text(displayPath)
                .lineLimit(1)
                .truncationMode(.middle)
                .foregroundColor(isSelected ? Color(NSColor.selectedMenuItemTextColor) : Color(NSColor.labelColor))
            Spacer()
        }
        .padding(.horizontal, 14)
        .padding(.vertical, 4)
        .listRowInsets(EdgeInsets())
        .background(isSelected ? Color(NSColor.controlAccentColor) : Color.clear)
    }
}

private struct KeyCatcher: NSViewRepresentable {
    let onMoveUp: () -> Void
    let onMoveDown: () -> Void
    let onCancel: () -> Void
    let onSubmit: () -> Void

    func makeNSView(context: Context) -> NSView {
        let view = KeyCatcherView()
        view.onMoveUp = onMoveUp
        view.onMoveDown = onMoveDown
        view.onCancel = onCancel
        view.onSubmit = onSubmit
        return view
    }

    func updateNSView(_ nsView: NSView, context: Context) {
        if let view = nsView as? KeyCatcherView {
            view.onMoveUp = onMoveUp
            view.onMoveDown = onMoveDown
            view.onCancel = onCancel
            view.onSubmit = onSubmit
        }
    }
}

private final class KeyCatcherView: NSView {
    var onMoveUp: () -> Void = {}
    var onMoveDown: () -> Void = {}
    var onCancel: () -> Void = {}
    var onSubmit: () -> Void = {}

    private var monitor: Any?

    override func viewDidMoveToWindow() {
        super.viewDidMoveToWindow()
        if window != nil, monitor == nil {
            monitor = NSEvent.addLocalMonitorForEvents(matching: .keyDown) { [weak self] event in
                guard let self = self else { return event }
                if event.keyCode == 53 { // esc
                    self.onCancel()
                    return nil
                }
                if event.keyCode == 126 { // up arrow
                    self.onMoveUp()
                    return nil
                }
                if event.keyCode == 125 { // down arrow
                    self.onMoveDown()
                    return nil
                }
                if event.modifierFlags.contains(.control) {
                    if let chars = event.charactersIgnoringModifiers {
                        if chars == "p" || chars == "k" { self.onMoveUp(); return nil }
                        if chars == "n" || chars == "j" { self.onMoveDown(); return nil }
                    }
                }
                return event
            }
        }
        if window == nil, let monitor = monitor {
            NSEvent.removeMonitor(monitor)
            self.monitor = nil
        }
    }

    deinit {
        if let monitor = monitor {
            NSEvent.removeMonitor(monitor)
        }
    }
}
