import SwiftUI
import AppKit

struct MarkdownEditor: NSViewRepresentable {
    @EnvironmentObject var appState: AppState

    func makeNSView(context: Context) -> NSScrollView {
        let scrollView = NSScrollView()
        scrollView.hasVerticalScroller = true
        scrollView.hasHorizontalScroller = true
        scrollView.borderType = .noBorder
        scrollView.drawsBackground = true
        scrollView.backgroundColor = Theme.editorBackground
        scrollView.autohidesScrollers = true
        scrollView.scrollerStyle = .overlay

        let textStorage = NSTextStorage()
        let layoutManager = NSLayoutManager()
        textStorage.addLayoutManager(layoutManager)

        let containerSize = NSSize(width: Theme.containerWidth, height: CGFloat.greatestFiniteMagnitude)
        let textContainer = NSTextContainer(size: containerSize)
        textContainer.widthTracksTextView = false
        textContainer.lineFragmentPadding = 0
        layoutManager.addTextContainer(textContainer)

        let textView = EditorTextView(frame: .zero, textContainer: textContainer)
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
        textView.isHorizontallyResizable = true
        textView.isVerticallyResizable = true
        textView.minSize = NSSize(width: 0, height: 0)
        textView.maxSize = NSSize(width: CGFloat.greatestFiniteMagnitude, height: CGFloat.greatestFiniteMagnitude)
        textView.autoresizingMask = []
        textView.textContainerInset = NSSize(
            width: Theme.editorHorizontalPadding,
            height: Theme.editorVerticalPadding
        )
        textView.backgroundColor = Theme.editorBackground
        textView.drawsBackground = true
        textView.insertionPointColor = Theme.bodyColor
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

        let visibleWidth = scrollView.contentSize.width
        if visibleWidth > 0 {
            let totalContentWidth = Theme.containerWidth + 2 * Theme.editorHorizontalPadding
            let frameWidth = max(visibleWidth, totalContentWidth)
            let extraSide = max(0, (frameWidth - totalContentWidth) / 2)
            let inset = Theme.editorHorizontalPadding + extraSide
            textView.textContainerInset = NSSize(width: inset, height: Theme.editorVerticalPadding)
            textView.frame.size.width = frameWidth
            textView.frame.size.height = max(textView.frame.size.height, scrollView.contentSize.height)
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
        if !appState.vimEnabled {
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
