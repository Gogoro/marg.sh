import AppKit

enum Theme {
    static let bodyFontSize: CGFloat = 16
    static let codeFontSize: CGFloat = 14

    static let lineHeightMultiple: CGFloat = 1.45
    static let bodyParagraphSpacing: CGFloat = 8

    static let maxContentWidth: CGFloat = 720
    static let editorHorizontalPadding: CGFloat = 32
    static let editorVerticalPadding: CGFloat = 40

    static let bodyFont: NSFont = serifFont(size: bodyFontSize, weight: .regular)
    static let boldFont: NSFont = serifFont(size: bodyFontSize, weight: .bold)
    static let italicFont: NSFont = {
        let base = serifFont(size: bodyFontSize, weight: .regular)
        let descriptor = base.fontDescriptor.withSymbolicTraits(.italic)
        return NSFont(descriptor: descriptor, size: bodyFontSize) ?? base
    }()
    static let monoFont: NSFont = NSFont.monospacedSystemFont(ofSize: codeFontSize, weight: .regular)

    static func headingFont(level: Int) -> NSFont {
        let size: CGFloat
        switch level {
        case 1: size = 30
        case 2: size = 25
        case 3: size = 21
        case 4: size = 19
        case 5: size = 18
        default: size = 17
        }
        return serifFont(size: size, weight: .semibold)
    }

    static let bodyColor = NSColor.labelColor
    static let secondaryColor = NSColor.secondaryLabelColor
    static let mutedColor = NSColor.tertiaryLabelColor
    static let codeColor = NSColor(srgbRed: 0.78, green: 0.43, blue: 0.55, alpha: 1)
    static let linkColor = NSColor.linkColor
    static let codeBlockBackground: NSColor = {
        return NSColor.textBackgroundColor.blended(
            withFraction: 0.06,
            of: .secondaryLabelColor
        ) ?? NSColor.controlBackgroundColor
    }()

    private static func serifFont(size: CGFloat, weight: NSFont.Weight) -> NSFont {
        let base = NSFont.systemFont(ofSize: size, weight: weight)
        if let descriptor = base.fontDescriptor.withDesign(.serif),
           let serif = NSFont(descriptor: descriptor, size: size) {
            return serif
        }
        return base
    }
}
