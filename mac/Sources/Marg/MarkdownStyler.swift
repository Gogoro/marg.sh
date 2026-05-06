import AppKit

enum MarkdownStyler {
    private static let asterisk: unichar = 0x2A
    private static let underscore: unichar = 0x5F
    private static let backtick: unichar = 0x60
    private static let openBracket: unichar = 0x5B
    private static let closeBracket: unichar = 0x5D
    private static let openParen: unichar = 0x28
    private static let closeParen: unichar = 0x29

    static func apply(to storage: NSTextStorage) {
        let nsText = storage.string as NSString
        storage.beginEditing()
        defer { storage.endEditing() }

        let fullRange = NSRange(location: 0, length: storage.length)
        storage.setAttributes(bodyAttributes(), range: fullRange)

        var lineStart = 0
        var insideCodeBlock = false
        var fenceContentStart: Int? = nil
        var fenceLanguage: String? = nil

        while lineStart < nsText.length {
            let lineRange = nsText.lineRange(for: NSRange(location: lineStart, length: 0))
            let contentRange = stripTrailingNewline(in: nsText, range: lineRange)
            let line = nsText.substring(with: contentRange)
            let trimmed = line.trimmingCharacters(in: .whitespaces)

            if trimmed.hasPrefix("```") {
                storage.setAttributes(codeFenceAttributes(), range: lineRange)
                if insideCodeBlock {
                    if let blockStart = fenceContentStart {
                        let blockRange = NSRange(location: blockStart, length: lineRange.location - blockStart)
                        CodeHighlighter.highlight(in: storage, range: blockRange, languageHint: fenceLanguage)
                    }
                    fenceContentStart = nil
                    fenceLanguage = nil
                } else {
                    let hint = String(trimmed.dropFirst(3)).trimmingCharacters(in: .whitespaces).lowercased()
                    fenceLanguage = hint.isEmpty ? nil : hint
                    fenceContentStart = NSMaxRange(lineRange)
                }
                insideCodeBlock.toggle()
            } else if insideCodeBlock {
                storage.setAttributes(codeBlockAttributes(), range: lineRange)
            } else if let level = headingLevel(of: line) {
                let attrs = headingAttributes(level: level)
                storage.setAttributes(attrs, range: lineRange)
                applyInlineStyles(in: storage, source: nsText, range: contentRange, baseAttrs: attrs)
                muteLeadingHashes(storage: storage, source: nsText, range: contentRange, level: level)
            } else if line.hasPrefix(">") {
                let attrs = quoteAttributes()
                storage.setAttributes(attrs, range: lineRange)
                applyInlineStyles(in: storage, source: nsText, range: contentRange, baseAttrs: attrs)
            } else {
                let attrs = bodyAttributes()
                applyInlineStyles(in: storage, source: nsText, range: contentRange, baseAttrs: attrs)
                muteLeadingListMarker(storage: storage, source: nsText, range: contentRange)
            }

            lineStart = NSMaxRange(lineRange)
        }
    }

    static func bodyAttributes() -> [NSAttributedString.Key: Any] {
        return [
            .font: Theme.bodyFont,
            .foregroundColor: Theme.bodyColor,
            .paragraphStyle: bodyParagraphStyle()
        ]
    }

    static func headingAttributes(level: Int) -> [NSAttributedString.Key: Any] {
        return [
            .font: Theme.headingFont(level: level),
            .foregroundColor: Theme.bodyColor,
            .paragraphStyle: headingParagraphStyle(level: level)
        ]
    }

    static func quoteAttributes() -> [NSAttributedString.Key: Any] {
        return [
            .font: Theme.italicFont,
            .foregroundColor: Theme.secondaryColor,
            .paragraphStyle: quoteParagraphStyle()
        ]
    }

    static func codeFenceAttributes() -> [NSAttributedString.Key: Any] {
        return [
            .font: Theme.monoFont,
            .foregroundColor: Theme.mutedColor,
            .paragraphStyle: codeFenceParagraphStyle()
        ]
    }

    static func codeBlockAttributes() -> [NSAttributedString.Key: Any] {
        return [
            .font: Theme.monoFont,
            .foregroundColor: Theme.bodyColor,
            .paragraphStyle: codeBlockParagraphStyle()
        ]
    }

    private static func bodyParagraphStyle() -> NSParagraphStyle {
        let style = NSMutableParagraphStyle()
        style.lineHeightMultiple = Theme.lineHeightMultiple
        style.paragraphSpacing = Theme.bodyParagraphSpacing
        return style
    }

    private static func headingParagraphStyle(level: Int) -> NSParagraphStyle {
        let style = NSMutableParagraphStyle()
        style.lineHeightMultiple = 1.15
        style.paragraphSpacingBefore = level == 1 ? 24 : 16
        style.paragraphSpacing = 8
        return style
    }

    private static func quoteParagraphStyle() -> NSParagraphStyle {
        let style = NSMutableParagraphStyle()
        style.headIndent = 18
        style.firstLineHeadIndent = 0
        style.lineHeightMultiple = Theme.lineHeightMultiple
        style.paragraphSpacing = Theme.bodyParagraphSpacing
        return style
    }

    private static func codeFenceParagraphStyle() -> NSParagraphStyle {
        let style = NSMutableParagraphStyle()
        style.lineHeightMultiple = 1.25
        style.paragraphSpacingBefore = 6
        style.lineBreakMode = .byCharWrapping
        return style
    }

    private static func codeBlockParagraphStyle() -> NSParagraphStyle {
        let style = NSMutableParagraphStyle()
        style.lineHeightMultiple = 1.3
        style.lineBreakMode = .byCharWrapping
        return style
    }

    private static func stripTrailingNewline(in text: NSString, range: NSRange) -> NSRange {
        var length = range.length
        while length > 0 {
            let last = text.character(at: range.location + length - 1)
            if last == 0x0A || last == 0x0D {
                length -= 1
            } else {
                break
            }
        }
        return NSRange(location: range.location, length: length)
    }

    private static func headingLevel(of line: String) -> Int? {
        var index = line.startIndex
        while index < line.endIndex && (line[index] == " " || line[index] == "\t") {
            index = line.index(after: index)
        }
        var hashCount = 0
        while index < line.endIndex && line[index] == "#" {
            hashCount += 1
            index = line.index(after: index)
        }
        guard hashCount >= 1 && hashCount <= 6 else { return nil }
        guard index < line.endIndex, line[index] == " " || line[index] == "\t" else { return nil }
        return hashCount
    }

    private static func muteLeadingHashes(
        storage: NSTextStorage,
        source: NSString,
        range: NSRange,
        level: Int
    ) {
        let endIndex = NSMaxRange(range)
        var index = range.location
        while index < endIndex {
            let c = source.character(at: index)
            if c == 0x20 || c == 0x09 { index += 1 } else { break }
        }
        let hashesEnd = index + level
        guard hashesEnd <= endIndex else { return }
        storage.addAttribute(
            .foregroundColor,
            value: Theme.mutedColor,
            range: NSRange(location: index, length: level)
        )
    }

    private static func muteLeadingListMarker(
        storage: NSTextStorage,
        source: NSString,
        range: NSRange
    ) {
        let endIndex = NSMaxRange(range)
        var index = range.location
        while index < endIndex {
            let c = source.character(at: index)
            if c == 0x20 || c == 0x09 { index += 1 } else { break }
        }
        guard index < endIndex else { return }
        let first = source.character(at: index)
        let isBullet = (first == 0x2D || first == 0x2A || first == 0x2B) // - * +
        let isNumber = (first >= 0x30 && first <= 0x39)
        var markerEnd = index

        if isBullet, index + 1 < endIndex, source.character(at: index + 1) == 0x20 {
            markerEnd = index + 2
        } else if isNumber {
            var scan = index
            while scan < endIndex {
                let ch = source.character(at: scan)
                if ch >= 0x30 && ch <= 0x39 { scan += 1 } else { break }
            }
            if scan < endIndex && source.character(at: scan) == 0x2E,
               scan + 1 < endIndex && source.character(at: scan + 1) == 0x20 {
                markerEnd = scan + 2
            }
        }

        if markerEnd > index {
            storage.addAttribute(
                .foregroundColor,
                value: Theme.secondaryColor,
                range: NSRange(location: index, length: markerEnd - index)
            )
        }
    }

    private static func applyInlineStyles(
        in storage: NSTextStorage,
        source: NSString,
        range: NSRange,
        baseAttrs: [NSAttributedString.Key: Any]
    ) {
        let endIndex = NSMaxRange(range)
        var i = range.location
        while i < endIndex {
            let c = source.character(at: i)

            if c == asterisk, i + 1 < endIndex, source.character(at: i + 1) == asterisk {
                if let close = findDoubleAsterisk(in: source, from: i + 2, end: endIndex) {
                    let span = NSRange(location: i, length: (close + 2) - i)
                    var attrs = baseAttrs
                    attrs[.font] = Theme.boldFont
                    storage.setAttributes(attrs, range: span)
                    storage.addAttribute(.foregroundColor, value: Theme.mutedColor, range: NSRange(location: i, length: 2))
                    storage.addAttribute(.foregroundColor, value: Theme.mutedColor, range: NSRange(location: close, length: 2))
                    i = close + 2
                    continue
                }
            }

            if c == asterisk || c == underscore {
                if let close = findUnit(in: source, from: i + 1, end: endIndex, target: c), close > i + 1 {
                    let span = NSRange(location: i, length: (close + 1) - i)
                    var attrs = baseAttrs
                    attrs[.font] = Theme.italicFont
                    storage.setAttributes(attrs, range: span)
                    storage.addAttribute(.foregroundColor, value: Theme.mutedColor, range: NSRange(location: i, length: 1))
                    storage.addAttribute(.foregroundColor, value: Theme.mutedColor, range: NSRange(location: close, length: 1))
                    i = close + 1
                    continue
                }
            }

            if c == backtick {
                if let close = findUnit(in: source, from: i + 1, end: endIndex, target: backtick), close > i {
                    let span = NSRange(location: i, length: (close + 1) - i)
                    var attrs = baseAttrs
                    attrs[.font] = Theme.monoFont
                    attrs[.foregroundColor] = Theme.codeColor
                    storage.setAttributes(attrs, range: span)
                    i = close + 1
                    continue
                }
            }

            if c == openBracket {
                if let bracketClose = findUnit(in: source, from: i + 1, end: endIndex, target: closeBracket),
                   bracketClose + 1 < endIndex,
                   source.character(at: bracketClose + 1) == openParen,
                   let parenClose = findUnit(in: source, from: bracketClose + 2, end: endIndex, target: closeParen) {
                    let span = NSRange(location: i, length: (parenClose + 1) - i)
                    var attrs = baseAttrs
                    attrs[.foregroundColor] = Theme.linkColor
                    attrs[.underlineStyle] = NSUnderlineStyle.single.rawValue
                    storage.setAttributes(attrs, range: span)
                    i = parenClose + 1
                    continue
                }
            }

            i += 1
        }
    }

    private static func findUnit(in source: NSString, from start: Int, end: Int, target: unichar) -> Int? {
        var i = start
        while i < end {
            if source.character(at: i) == target { return i }
            i += 1
        }
        return nil
    }

    private static func findDoubleAsterisk(in source: NSString, from start: Int, end: Int) -> Int? {
        var i = start
        while i + 1 < end {
            if source.character(at: i) == asterisk, source.character(at: i + 1) == asterisk {
                return i
            }
            i += 1
        }
        return nil
    }
}
