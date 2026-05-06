import SwiftUI
import AppKit

struct MarkdownEditor: NSViewRepresentable {
    @EnvironmentObject var appState: AppState

    func makeNSView(context: Context) -> NSScrollView {
        let scrollView = NSScrollView()
        scrollView.hasVerticalScroller = true
        scrollView.hasHorizontalScroller = false
        scrollView.borderType = .noBorder
        scrollView.drawsBackground = false
        scrollView.autohidesScrollers = true

        let textView = EditorTextView()
        textView.delegate = context.coordinator
        textView.coordinator = context.coordinator
        textView.appState = appState
        textView.allowsUndo = true
        textView.isRichText = false
        textView.usesFontPanel = false
        textView.isAutomaticQuoteSubstitutionEnabled = false
        textView.isAutomaticDashSubstitutionEnabled = false
        textView.isAutomaticTextReplacementEnabled = false
        textView.isAutomaticSpellingCorrectionEnabled = false
        textView.isContinuousSpellCheckingEnabled = false
        textView.isAutomaticLinkDetectionEnabled = false
        textView.isAutomaticDataDetectionEnabled = false
        textView.smartInsertDeleteEnabled = false
        textView.isHorizontallyResizable = false
        textView.isVerticallyResizable = true
        textView.minSize = NSSize(width: 0, height: 0)
        textView.maxSize = NSSize(width: CGFloat.greatestFiniteMagnitude, height: CGFloat.greatestFiniteMagnitude)
        textView.autoresizingMask = [.width]
        textView.textContainer?.widthTracksTextView = true
        textView.textContainer?.lineFragmentPadding = 0
        textView.textContainerInset = NSSize(
            width: Theme.editorHorizontalPadding,
            height: Theme.editorVerticalPadding
        )
        textView.backgroundColor = NSColor.textBackgroundColor
        textView.drawsBackground = true
        textView.insertionPointColor = NSColor.controlAccentColor
        textView.font = Theme.bodyFont
        textView.textColor = Theme.bodyColor

        scrollView.documentView = textView
        context.coordinator.textView = textView

        if !appState.text.isEmpty {
            textView.string = appState.text
            if let storage = textView.textStorage {
                MarkdownStyler.apply(to: storage)
            }
        }

        return scrollView
    }

    func updateNSView(_ scrollView: NSScrollView, context: Context) {
        guard let textView = scrollView.documentView as? EditorTextView else { return }
        let coordinator = context.coordinator

        textView.appState = appState

        if appState.text != textView.string && !coordinator.isUpdatingFromTextView {
            coordinator.isApplyingExternalText = true
            textView.string = appState.text
            if let storage = textView.textStorage {
                MarkdownStyler.apply(to: storage)
            }
            coordinator.isApplyingExternalText = false
        }

        let scrollWidth = scrollView.contentSize.width
        if scrollWidth > 0 {
            let availableForText = scrollWidth
            let textWidth = min(Theme.maxContentWidth, max(0, availableForText - 2 * Theme.editorHorizontalPadding))
            let totalSidePad = max(Theme.editorHorizontalPadding, (availableForText - textWidth) / 2)
            textView.textContainerInset = NSSize(width: totalSidePad, height: Theme.editorVerticalPadding)
        }
    }

    func makeCoordinator() -> Coordinator {
        Coordinator(parent: self)
    }

    final class Coordinator: NSObject, NSTextViewDelegate {
        var parent: MarkdownEditor
        weak var textView: NSTextView?
        var isApplyingExternalText = false
        var isUpdatingFromTextView = false
        private var styleDebounce: DispatchWorkItem?

        init(parent: MarkdownEditor) {
            self.parent = parent
        }

        func textDidChange(_ notification: Notification) {
            guard let textView = notification.object as? NSTextView else { return }
            guard !isApplyingExternalText else { return }

            isUpdatingFromTextView = true
            parent.appState.text = textView.string
            parent.appState.markDirty()
            isUpdatingFromTextView = false

            scheduleRestyle(for: textView)
        }

        private func scheduleRestyle(for textView: NSTextView) {
            styleDebounce?.cancel()
            let work = DispatchWorkItem { [weak textView] in
                guard let storage = textView?.textStorage else { return }
                MarkdownStyler.apply(to: storage)
            }
            styleDebounce = work
            DispatchQueue.main.asyncAfter(deadline: .now() + 0.2, execute: work)
        }
    }
}

final class EditorTextView: NSTextView {
    weak var coordinator: MarkdownEditor.Coordinator?
    var appState: AppState?

    private let vimHandler = VimKeyHandler()

    override func keyDown(with event: NSEvent) {
        guard let appState = appState else {
            super.keyDown(with: event)
            return
        }
        vimHandler.textView = self
        vimHandler.appState = appState
        if vimHandler.handle(event: event) {
            return
        }
        super.keyDown(with: event)
    }
}
