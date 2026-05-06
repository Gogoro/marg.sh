import AppKit
import SwiftUI

enum Theme {
    static let bodyFontSize: CGFloat = 17
    static let codeFontSize: CGFloat = 14.5

    static let lineHeightMultiple: CGFloat = 1.55
    static let bodyParagraphSpacing: CGFloat = 12

    static let maxContentWidth: CGFloat = 720
    static let editorHorizontalPadding: CGFloat = 56
    static let editorVerticalPadding: CGFloat = 72
    static let sidebarWidth: CGFloat = 240
    static let statusBarHeight: CGFloat = 28
    static let titleBarInset: CGFloat = 36

    static let bodyFont: NSFont = NSFont.systemFont(ofSize: bodyFontSize, weight: .regular)
    static let boldFont: NSFont = NSFont.systemFont(ofSize: bodyFontSize, weight: .semibold)
    static let italicFont: NSFont = {
        let base = NSFont.systemFont(ofSize: bodyFontSize, weight: .regular)
        let descriptor = base.fontDescriptor.withSymbolicTraits(.italic)
        return NSFont(descriptor: descriptor, size: bodyFontSize) ?? base
    }()
    static let monoFont: NSFont = NSFont.monospacedSystemFont(ofSize: codeFontSize, weight: .regular)

    static func headingFont(level: Int) -> NSFont {
        let size: CGFloat
        let weight: NSFont.Weight
        switch level {
        case 1: size = 34; weight = .bold
        case 2: size = 26; weight = .semibold
        case 3: size = 21; weight = .semibold
        case 4: size = 18; weight = .semibold
        case 5: size = 17; weight = .semibold
        default: size = 16; weight = .semibold
        }
        return NSFont.systemFont(ofSize: size, weight: weight)
    }

    // NSColors for the NSAttributedString side.
    static let bodyColor = NSColor(srgbRed: 0.10, green: 0.10, blue: 0.10, alpha: 1)
    static let secondaryColor = NSColor(srgbRed: 0.47, green: 0.47, blue: 0.45, alpha: 1)
    static let mutedColor = NSColor(srgbRed: 0.78, green: 0.78, blue: 0.76, alpha: 1)
    static let codeColor = NSColor(srgbRed: 0.78, green: 0.30, blue: 0.45, alpha: 1)
    static let linkColor = NSColor(srgbRed: 0.15, green: 0.39, blue: 0.92, alpha: 1)
    static let editorBackground = NSColor.white

    static let syntaxKeywordColor = NSColor(srgbRed: 0.36, green: 0.30, blue: 0.63, alpha: 1)
    static let syntaxTypeColor = NSColor(srgbRed: 0.10, green: 0.42, blue: 0.48, alpha: 1)
    static let syntaxStringColor = NSColor(srgbRed: 0.37, green: 0.48, blue: 0.14, alpha: 1)
    static let syntaxNumberColor = NSColor(srgbRed: 0.63, green: 0.32, blue: 0.18, alpha: 1)
    static let syntaxCommentColor = NSColor(srgbRed: 0.55, green: 0.55, blue: 0.50, alpha: 1)

    // Color tokens for the SwiftUI side.
    static let editorBackgroundColor = Color.white
    static let sidebarBackgroundColor = Color(NSColor(srgbRed: 0.973, green: 0.969, blue: 0.961, alpha: 1))
    static let sidebarHoverColor = Color(NSColor(srgbRed: 0, green: 0, blue: 0, alpha: 0.045))
    static let sidebarSelectionColor = Color(NSColor(srgbRed: 0, green: 0, blue: 0, alpha: 0.075))
    static let dividerColor = Color(NSColor(srgbRed: 0.91, green: 0.91, blue: 0.89, alpha: 1))
    static let primaryTextColor = Color(NSColor(srgbRed: 0.10, green: 0.10, blue: 0.10, alpha: 1))
    static let secondaryTextColor = Color(NSColor(srgbRed: 0.47, green: 0.47, blue: 0.45, alpha: 1))
    static let mutedTextColor = Color(NSColor(srgbRed: 0.65, green: 0.65, blue: 0.62, alpha: 1))
    static let accentTextColor = Color(NSColor(srgbRed: 0.10, green: 0.10, blue: 0.10, alpha: 1))
    static let dirtyDotColor = Color(NSColor(srgbRed: 0.92, green: 0.55, blue: 0.20, alpha: 1))

    static let pickerOverlayColor = Color.black.opacity(0.18)
    static let pickerCardColor = Color.white
    static let pickerCardBorderColor = Color(NSColor(srgbRed: 0.86, green: 0.86, blue: 0.84, alpha: 1))
    static let pickerSelectionColor = Color(NSColor(srgbRed: 0.95, green: 0.95, blue: 0.93, alpha: 1))

    static let bodyTextSwiftUI = Font.system(size: 13, weight: .regular)
    static let sidebarFont = Font.system(size: 13, weight: .regular)
    static let sidebarHeaderFont = Font.system(size: 11, weight: .semibold).smallCaps()
    static let statusFont = Font.system(size: 11, weight: .regular)
    static let pickerInputFont = Font.system(size: 18, weight: .regular)
    static let pickerRowFont = Font.system(size: 13, weight: .regular)
}
