import SwiftUI
import AppKit

struct FuzzyPickerOverlay: View {
    @EnvironmentObject var appState: AppState
    @State private var query: String = ""
    @State private var cursor: Int = 0
    @FocusState private var inputFocused: Bool

    var body: some View {
        ZStack {
            Theme.pickerOverlayColor
                .ignoresSafeArea()
                .onTapGesture { close() }

            card
                .frame(width: 620, height: 460)
                .shadow(color: Color.black.opacity(0.18), radius: 32, x: 0, y: 16)
        }
        .background(KeyCatcher(
            onMoveUp: { moveCursor(-1) },
            onMoveDown: { moveCursor(1) },
            onCancel: { close() },
            onSubmit: { openSelection() }
        ))
        .onAppear { inputFocused = true }
        .transition(.opacity)
    }

    private var card: some View {
        VStack(spacing: 0) {
            HStack(spacing: 10) {
                Image(systemName: "magnifyingglass")
                    .foregroundColor(Theme.mutedTextColor)
                    .font(.system(size: 14, weight: .medium))
                TextField("type to filter", text: $query)
                    .textFieldStyle(.plain)
                    .font(Theme.pickerInputFont)
                    .focused($inputFocused)
                    .onSubmit { openSelection() }
                    .onChange(of: query) { _, _ in cursor = 0 }
                if !query.isEmpty {
                    Text("\(matches.count)")
                        .font(.system(size: 11))
                        .foregroundColor(Theme.mutedTextColor)
                }
            }
            .padding(.horizontal, 18)
            .padding(.vertical, 16)

            Rectangle()
                .fill(Theme.dividerColor)
                .frame(height: 1)

            ScrollViewReader { proxy in
                ScrollView {
                    LazyVStack(spacing: 0) {
                        ForEach(Array(matches.enumerated()), id: \.element.url) { index, item in
                            FuzzyPickerRow(displayPath: item.display, isSelected: index == cursor)
                                .id(item.url)
                                .onTapGesture { openItem(item.url) }
                        }
                    }
                    .padding(.vertical, 6)
                }
                .onChange(of: cursor) { _, newValue in
                    guard newValue < matches.count else { return }
                    proxy.scrollTo(matches[newValue].url, anchor: .center)
                }
            }
        }
        .background(Theme.pickerCardColor)
        .clipShape(RoundedRectangle(cornerRadius: 12))
        .overlay(
            RoundedRectangle(cornerRadius: 12)
                .stroke(Theme.pickerCardBorderColor, lineWidth: 1)
        )
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
    }
}

private struct FuzzyPickerRow: View {
    let displayPath: String
    let isSelected: Bool

    var body: some View {
        HStack(spacing: 8) {
            Image(systemName: "doc.text")
                .font(.system(size: 11))
                .foregroundColor(isSelected ? Theme.primaryTextColor : Theme.mutedTextColor)
                .frame(width: 14)
            Text(displayPath)
                .font(Theme.pickerRowFont)
                .lineLimit(1)
                .truncationMode(.middle)
                .foregroundColor(Theme.primaryTextColor)
            Spacer()
        }
        .padding(.horizontal, 14)
        .padding(.vertical, 7)
        .background(isSelected ? Theme.pickerSelectionColor : Color.clear)
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
                if event.keyCode == 53 { self.onCancel(); return nil }
                if event.keyCode == 126 { self.onMoveUp(); return nil }
                if event.keyCode == 125 { self.onMoveDown(); return nil }
                if event.modifierFlags.contains(.control), let chars = event.charactersIgnoringModifiers {
                    if chars == "p" || chars == "k" { self.onMoveUp(); return nil }
                    if chars == "n" || chars == "j" { self.onMoveDown(); return nil }
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
