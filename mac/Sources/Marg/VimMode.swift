import AppKit
import Foundation

enum VimMode {
    case normal
    case insert
    case visual
    case commandLine
}

final class VimKeyHandler {
    weak var textView: NSTextView?
    weak var appState: AppState?

    private var pending: String?

    func handle(event: NSEvent) -> Bool {
        guard let appState = appState, let textView = textView else { return false }

        if event.keyCode == 53 { // escape
            pending = nil
            switch appState.vimMode {
            case .insert, .visual:
                appState.vimMode = .normal
                let range = textView.selectedRange()
                textView.setSelectedRange(NSRange(location: range.location, length: 0))
            case .commandLine:
                appState.commandLineBuffer = ""
                appState.vimMode = .normal
            case .normal:
                break
            }
            return true
        }

        if appState.vimMode != .commandLine && event.modifierFlags.contains(.command) {
            return false
        }

        switch appState.vimMode {
        case .insert:
            return false
        case .normal:
            return handleNormalMode(event: event, textView: textView, appState: appState)
        case .visual:
            return handleVisualMode(event: event, textView: textView, appState: appState)
        case .commandLine:
            return handleCommandLine(event: event, appState: appState)
        }
    }

    private func handleNormalMode(event: NSEvent, textView: NSTextView, appState: AppState) -> Bool {
        let chars = event.charactersIgnoringModifiers ?? ""

        if let active = pending {
            pending = nil
            switch (active, chars) {
            case ("d", "d"):
                deleteLine(in: textView)
                return true
            case ("y", "y"):
                yankLine(in: textView)
                return true
            case ("g", "g"):
                textView.moveToBeginningOfDocument(nil)
                return true
            default:
                break
            }
        }

        if event.modifierFlags.contains(.control), chars == "r" {
            textView.undoManager?.redo()
            return true
        }

        switch chars {
        case "h": textView.moveLeft(nil); return true
        case "l": textView.moveRight(nil); return true
        case "j": textView.moveDown(nil); return true
        case "k": textView.moveUp(nil); return true
        case "0": textView.moveToBeginningOfLine(nil); return true
        case "$": textView.moveToEndOfLine(nil); return true
        case "G": textView.moveToEndOfDocument(nil); return true
        case "g": pending = "g"; return true
        case "w": textView.moveWordForward(nil); return true
        case "b": textView.moveWordBackward(nil); return true
        case "i": appState.vimMode = .insert; return true
        case "I":
            textView.moveToBeginningOfLine(nil)
            appState.vimMode = .insert
            return true
        case "a":
            textView.moveRight(nil)
            appState.vimMode = .insert
            return true
        case "A":
            textView.moveToEndOfLine(nil)
            appState.vimMode = .insert
            return true
        case "o":
            openLineBelow(in: textView)
            appState.vimMode = .insert
            return true
        case "O":
            openLineAbove(in: textView)
            appState.vimMode = .insert
            return true
        case "x":
            deleteCharForward(in: textView)
            return true
        case "d": pending = "d"; return true
        case "y": pending = "y"; return true
        case "p":
            pasteAfter(in: textView)
            return true
        case "P":
            pasteBefore(in: textView)
            return true
        case "v": appState.vimMode = .visual; return true
        case ":":
            appState.vimMode = .commandLine
            appState.commandLineBuffer = ""
            return true
        case "u":
            textView.undoManager?.undo()
            return true
        default:
            switch event.keyCode {
            case 123: textView.moveLeft(nil); return true
            case 124: textView.moveRight(nil); return true
            case 125: textView.moveDown(nil); return true
            case 126: textView.moveUp(nil); return true
            default: return true
            }
        }
    }

    private func handleVisualMode(event: NSEvent, textView: NSTextView, appState: AppState) -> Bool {
        let chars = event.charactersIgnoringModifiers ?? ""

        switch chars {
        case "h": textView.moveLeftAndModifySelection(nil); return true
        case "l": textView.moveRightAndModifySelection(nil); return true
        case "j": textView.moveDownAndModifySelection(nil); return true
        case "k": textView.moveUpAndModifySelection(nil); return true
        case "0": textView.moveToBeginningOfLineAndModifySelection(nil); return true
        case "$": textView.moveToEndOfLineAndModifySelection(nil); return true
        case "w": textView.moveWordForwardAndModifySelection(nil); return true
        case "b": textView.moveWordBackwardAndModifySelection(nil); return true
        case "G": textView.moveToEndOfDocumentAndModifySelection(nil); return true
        case "y":
            yankSelection(in: textView)
            collapseSelection(in: textView)
            appState.vimMode = .normal
            return true
        case "d":
            deleteSelection(in: textView)
            appState.vimMode = .normal
            return true
        default:
            switch event.keyCode {
            case 123: textView.moveLeftAndModifySelection(nil); return true
            case 124: textView.moveRightAndModifySelection(nil); return true
            case 125: textView.moveDownAndModifySelection(nil); return true
            case 126: textView.moveUpAndModifySelection(nil); return true
            default: return true
            }
        }
    }

    private func handleCommandLine(event: NSEvent, appState: AppState) -> Bool {
        if event.keyCode == 36 { // return
            appState.runCommandLine(appState.commandLineBuffer)
            return true
        }
        if event.keyCode == 51 { // delete (backspace)
            if !appState.commandLineBuffer.isEmpty {
                appState.commandLineBuffer.removeLast()
            } else {
                appState.vimMode = .normal
            }
            return true
        }
        if let chars = event.charactersIgnoringModifiers, !chars.isEmpty {
            for ch in chars where !ch.isNewline {
                appState.commandLineBuffer.append(ch)
            }
            return true
        }
        return true
    }

    private func deleteLine(in textView: NSTextView) {
        guard let storage = textView.textStorage else { return }
        let text = storage.string as NSString
        let cursor = textView.selectedRange()
        let lineRange = text.lineRange(for: NSRange(location: cursor.location, length: 0))
        let lineText = text.substring(with: lineRange)
        copyToPasteboard(lineText)
        if textView.shouldChangeText(in: lineRange, replacementString: "") {
            textView.replaceCharacters(in: lineRange, with: "")
            textView.didChangeText()
        }
    }

    private func yankLine(in textView: NSTextView) {
        guard let storage = textView.textStorage else { return }
        let text = storage.string as NSString
        let cursor = textView.selectedRange()
        let lineRange = text.lineRange(for: NSRange(location: cursor.location, length: 0))
        copyToPasteboard(text.substring(with: lineRange))
    }

    private func openLineBelow(in textView: NSTextView) {
        guard let storage = textView.textStorage else { return }
        let text = storage.string as NSString
        let cursor = textView.selectedRange()
        let lineRange = text.lineRange(for: NSRange(location: cursor.location, length: 0))
        let insertAt = NSMaxRange(lineRange)
        let suffix = text.substring(with: lineRange)
        let inserted: String = suffix.hasSuffix("\n") ? "\n" : "\n"
        let target = NSRange(location: insertAt, length: 0)
        if textView.shouldChangeText(in: target, replacementString: inserted) {
            textView.replaceCharacters(in: target, with: inserted)
            textView.didChangeText()
            let newCursor = insertAt + (inserted as NSString).length
            textView.setSelectedRange(NSRange(location: newCursor, length: 0))
        }
    }

    private func openLineAbove(in textView: NSTextView) {
        guard let storage = textView.textStorage else { return }
        let text = storage.string as NSString
        let cursor = textView.selectedRange()
        let lineRange = text.lineRange(for: NSRange(location: cursor.location, length: 0))
        let insertAt = lineRange.location
        let inserted = "\n"
        let target = NSRange(location: insertAt, length: 0)
        if textView.shouldChangeText(in: target, replacementString: inserted) {
            textView.replaceCharacters(in: target, with: inserted)
            textView.didChangeText()
            textView.setSelectedRange(NSRange(location: insertAt, length: 0))
        }
    }

    private func deleteCharForward(in textView: NSTextView) {
        guard let storage = textView.textStorage else { return }
        let cursor = textView.selectedRange()
        guard cursor.location < storage.length else { return }
        let target = NSRange(location: cursor.location, length: 1)
        if textView.shouldChangeText(in: target, replacementString: "") {
            textView.replaceCharacters(in: target, with: "")
            textView.didChangeText()
        }
    }

    private func pasteAfter(in textView: NSTextView) {
        guard let str = NSPasteboard.general.string(forType: .string) else { return }
        guard let storage = textView.textStorage else { return }
        let text = storage.string as NSString
        let cursor = textView.selectedRange()
        if str.hasSuffix("\n") {
            let lineRange = text.lineRange(for: NSRange(location: cursor.location, length: 0))
            let insertAt = NSMaxRange(lineRange)
            let target = NSRange(location: insertAt, length: 0)
            if textView.shouldChangeText(in: target, replacementString: str) {
                textView.replaceCharacters(in: target, with: str)
                textView.didChangeText()
                textView.setSelectedRange(NSRange(location: insertAt, length: 0))
            }
        } else {
            let insertAt = min(cursor.location + 1, storage.length)
            let target = NSRange(location: insertAt, length: 0)
            if textView.shouldChangeText(in: target, replacementString: str) {
                textView.replaceCharacters(in: target, with: str)
                textView.didChangeText()
            }
        }
    }

    private func pasteBefore(in textView: NSTextView) {
        guard let str = NSPasteboard.general.string(forType: .string) else { return }
        guard let storage = textView.textStorage else { return }
        let text = storage.string as NSString
        let cursor = textView.selectedRange()
        if str.hasSuffix("\n") {
            let lineRange = text.lineRange(for: NSRange(location: cursor.location, length: 0))
            let target = NSRange(location: lineRange.location, length: 0)
            if textView.shouldChangeText(in: target, replacementString: str) {
                textView.replaceCharacters(in: target, with: str)
                textView.didChangeText()
                textView.setSelectedRange(NSRange(location: lineRange.location, length: 0))
            }
        } else {
            let target = NSRange(location: cursor.location, length: 0)
            if textView.shouldChangeText(in: target, replacementString: str) {
                textView.replaceCharacters(in: target, with: str)
                textView.didChangeText()
            }
        }
    }

    private func yankSelection(in textView: NSTextView) {
        guard let storage = textView.textStorage else { return }
        let range = textView.selectedRange()
        guard range.length > 0 else { return }
        let text = storage.string as NSString
        copyToPasteboard(text.substring(with: range))
    }

    private func deleteSelection(in textView: NSTextView) {
        let range = textView.selectedRange()
        guard range.length > 0 else { return }
        if let storage = textView.textStorage {
            let text = storage.string as NSString
            copyToPasteboard(text.substring(with: range))
        }
        if textView.shouldChangeText(in: range, replacementString: "") {
            textView.replaceCharacters(in: range, with: "")
            textView.didChangeText()
        }
    }

    private func collapseSelection(in textView: NSTextView) {
        let range = textView.selectedRange()
        textView.setSelectedRange(NSRange(location: range.location, length: 0))
    }

    private func copyToPasteboard(_ string: String) {
        NSPasteboard.general.clearContents()
        NSPasteboard.general.setString(string, forType: .string)
    }
}
