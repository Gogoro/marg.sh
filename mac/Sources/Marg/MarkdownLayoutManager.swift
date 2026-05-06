import AppKit

extension NSAttributedString.Key {
    static let margCodeBlock = NSAttributedString.Key("margCodeBlock")
}

final class MarkdownLayoutManager: NSLayoutManager {
    override func drawBackground(forGlyphRange glyphsToShow: NSRange, at origin: NSPoint) {
        super.drawBackground(forGlyphRange: glyphsToShow, at: origin)

        guard let textStorage = textStorage else { return }
        let visibleCharRange = characterRange(forGlyphRange: glyphsToShow, actualGlyphRange: nil)

        textStorage.enumerateAttribute(.margCodeBlock, in: visibleCharRange, options: []) { value, range, _ in
            guard (value as? Bool) == true else { return }
            drawCodeBlockBackground(forCharRange: range, origin: origin)
        }
    }

    private func drawCodeBlockBackground(forCharRange charRange: NSRange, origin: NSPoint) {
        let glyphs = glyphRange(forCharacterRange: charRange, actualCharacterRange: nil)
        guard glyphs.length > 0 else { return }

        var union: NSRect?
        enumerateLineFragments(forGlyphRange: glyphs) { lineRect, _, _, _, _ in
            let translated = lineRect.offsetBy(dx: origin.x, dy: origin.y)
            union = union.map { $0.union(translated) } ?? translated
        }

        guard let rect = union else { return }
        let padded = NSRect(
            x: rect.minX - 14,
            y: rect.minY - 4,
            width: rect.width + 28,
            height: rect.height + 8
        )
        let path = NSBezierPath(roundedRect: padded, xRadius: 6, yRadius: 6)
        Theme.codeBlockBackground.setFill()
        path.fill()
    }
}
