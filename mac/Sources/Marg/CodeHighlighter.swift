import AppKit

enum CodeHighlighter {
    struct Language {
        let keywords: Set<String>
        let types: Set<String>
        let lineCommentTokens: [String]
        let stringDelimiters: Set<unichar>
    }

    static func highlight(in storage: NSTextStorage, range: NSRange, languageHint: String?) {
        guard let hint = languageHint, let language = lookup(hint) else { return }
        tokenize(in: storage, range: range, language: language)
    }

    private static func lookup(_ hint: String) -> Language? {
        switch hint {
        case "swift": return swift
        case "js", "javascript", "mjs", "cjs": return javascript
        case "ts", "typescript", "tsx", "jsx": return typescript
        case "py", "python": return python
        case "go", "golang": return go
        case "rust", "rs": return rust
        case "sh", "bash", "shell", "zsh": return bash
        case "json": return json
        case "yaml", "yml": return yaml
        default: return nil
        }
    }

    private static func tokenize(in storage: NSTextStorage, range: NSRange, language: Language) {
        let source = storage.string as NSString
        var index = range.location
        let end = NSMaxRange(range)

        while index < end {
            let character = source.character(at: index)

            if let length = matchLineCommentPrefix(at: index, end: end, source: source, language: language) {
                var scan = index + length
                while scan < end && source.character(at: scan) != newline {
                    scan += 1
                }
                paint(storage, range: NSRange(location: index, length: scan - index), color: Theme.syntaxCommentColor)
                index = scan
                continue
            }

            if language.stringDelimiters.contains(character) {
                let stringEnd = consumeString(source: source, from: index, end: end, delimiter: character)
                paint(storage, range: NSRange(location: index, length: stringEnd - index), color: Theme.syntaxStringColor)
                index = stringEnd
                continue
            }

            if isAsciiDigit(character) {
                let numberEnd = consumeNumber(source: source, from: index, end: end)
                paint(storage, range: NSRange(location: index, length: numberEnd - index), color: Theme.syntaxNumberColor)
                index = numberEnd
                continue
            }

            if isIdentifierStart(character) {
                let identifierEnd = consumeIdentifier(source: source, from: index, end: end)
                let word = source.substring(with: NSRange(location: index, length: identifierEnd - index))
                if language.keywords.contains(word) {
                    paint(storage, range: NSRange(location: index, length: identifierEnd - index), color: Theme.syntaxKeywordColor)
                } else if language.types.contains(word) {
                    paint(storage, range: NSRange(location: index, length: identifierEnd - index), color: Theme.syntaxTypeColor)
                }
                index = identifierEnd
                continue
            }

            index += 1
        }
    }

    private static func paint(_ storage: NSTextStorage, range: NSRange, color: NSColor) {
        guard range.length > 0 else { return }
        storage.addAttribute(.foregroundColor, value: color, range: range)
    }

    private static func matchLineCommentPrefix(
        at index: Int,
        end: Int,
        source: NSString,
        language: Language
    ) -> Int? {
        for token in language.lineCommentTokens {
            let tokenLength = token.utf16.count
            if index + tokenLength > end { continue }
            var matches = true
            for (offset, scalar) in token.utf16.enumerated() {
                if source.character(at: index + offset) != scalar {
                    matches = false
                    break
                }
            }
            if matches { return tokenLength }
        }
        return nil
    }

    private static func consumeString(source: NSString, from start: Int, end: Int, delimiter: unichar) -> Int {
        var scan = start + 1
        while scan < end {
            let character = source.character(at: scan)
            if character == backslash && scan + 1 < end {
                scan += 2
                continue
            }
            if character == delimiter {
                return scan + 1
            }
            if character == newline {
                return scan
            }
            scan += 1
        }
        return scan
    }

    private static func consumeNumber(source: NSString, from start: Int, end: Int) -> Int {
        var scan = start + 1
        while scan < end {
            let character = source.character(at: scan)
            if isAsciiDigit(character) || character == dot || character == underscore {
                scan += 1
            } else {
                break
            }
        }
        return scan
    }

    private static func consumeIdentifier(source: NSString, from start: Int, end: Int) -> Int {
        var scan = start + 1
        while scan < end {
            if isIdentifierContinue(source.character(at: scan)) {
                scan += 1
            } else {
                break
            }
        }
        return scan
    }

    private static let newline: unichar = 0x0A
    private static let backslash: unichar = 0x5C
    private static let dot: unichar = 0x2E
    private static let underscore: unichar = 0x5F
    private static let doubleQuote: unichar = 0x22
    private static let singleQuote: unichar = 0x27
    private static let backtick: unichar = 0x60

    private static func isAsciiDigit(_ c: unichar) -> Bool {
        return c >= 0x30 && c <= 0x39
    }

    private static func isAsciiLetter(_ c: unichar) -> Bool {
        return (c >= 0x41 && c <= 0x5A) || (c >= 0x61 && c <= 0x7A)
    }

    private static func isIdentifierStart(_ c: unichar) -> Bool {
        return isAsciiLetter(c) || c == underscore
    }

    private static func isIdentifierContinue(_ c: unichar) -> Bool {
        return isIdentifierStart(c) || isAsciiDigit(c)
    }

    static let swift = Language(
        keywords: [
            "associatedtype", "class", "deinit", "enum", "extension", "fileprivate", "func", "import",
            "init", "inout", "internal", "let", "open", "operator", "private", "protocol", "public",
            "rethrows", "static", "struct", "subscript", "typealias", "var",
            "break", "case", "continue", "default", "defer", "do", "else", "fallthrough", "for",
            "guard", "if", "in", "repeat", "return", "switch", "where", "while",
            "as", "catch", "false", "is", "nil", "throw", "throws", "true", "try", "self", "Self",
            "super", "async", "await", "actor", "some", "any", "lazy", "weak", "unowned", "final",
            "override", "mutating", "nonmutating", "convenience", "required", "indirect"
        ],
        types: [
            "String", "Substring", "Character",
            "Int", "Int8", "Int16", "Int32", "Int64",
            "UInt", "UInt8", "UInt16", "UInt32", "UInt64",
            "Double", "Float", "CGFloat", "Bool",
            "Array", "Dictionary", "Set", "Optional", "Result",
            "Void", "Any", "AnyObject",
            "URL", "Data", "Date",
            "NSRange", "NSRect", "NSPoint", "NSSize", "NSColor", "NSFont", "NSString",
            "NSTextStorage", "NSLayoutManager", "NSTextContainer", "NSAttributedString"
        ],
        lineCommentTokens: ["//"],
        stringDelimiters: [doubleQuote]
    )

    static let javascript = Language(
        keywords: [
            "var", "let", "const", "function", "return", "if", "else", "for", "while", "do",
            "break", "continue", "switch", "case", "default", "throw", "try", "catch", "finally",
            "new", "delete", "typeof", "instanceof", "in", "of", "this", "super",
            "class", "extends", "import", "from", "export", "as",
            "async", "await", "yield", "void",
            "true", "false", "null", "undefined"
        ],
        types: ["String", "Number", "Boolean", "Object", "Array", "Map", "Set", "Promise", "Symbol"],
        lineCommentTokens: ["//"],
        stringDelimiters: [doubleQuote, singleQuote, backtick]
    )

    static let typescript = Language(
        keywords: [
            "var", "let", "const", "function", "return", "if", "else", "for", "while", "do",
            "break", "continue", "switch", "case", "default", "throw", "try", "catch", "finally",
            "new", "delete", "typeof", "instanceof", "in", "of", "this", "super",
            "class", "extends", "implements", "interface", "type", "enum", "namespace",
            "import", "from", "export", "as", "public", "private", "protected", "readonly",
            "abstract", "static", "async", "await", "yield", "void",
            "true", "false", "null", "undefined"
        ],
        types: [
            "string", "number", "boolean", "object", "any", "unknown", "never",
            "Array", "Map", "Set", "Promise", "Record", "Partial", "Readonly", "Pick", "Omit"
        ],
        lineCommentTokens: ["//"],
        stringDelimiters: [doubleQuote, singleQuote, backtick]
    )

    static let python = Language(
        keywords: [
            "and", "as", "assert", "async", "await", "break", "class", "continue", "def", "del",
            "elif", "else", "except", "finally", "for", "from", "global", "if", "import", "in",
            "is", "lambda", "nonlocal", "not", "or", "pass", "raise", "return", "try", "while",
            "with", "yield", "True", "False", "None", "self"
        ],
        types: [
            "int", "str", "float", "bool", "list", "dict", "tuple", "set", "frozenset", "bytes",
            "bytearray", "object", "type", "Exception"
        ],
        lineCommentTokens: ["#"],
        stringDelimiters: [doubleQuote, singleQuote]
    )

    static let go = Language(
        keywords: [
            "break", "case", "chan", "const", "continue", "default", "defer", "else", "fallthrough",
            "for", "func", "go", "goto", "if", "import", "interface", "map", "package", "range",
            "return", "select", "struct", "switch", "type", "var",
            "true", "false", "nil", "iota"
        ],
        types: [
            "bool", "byte", "rune", "string", "error",
            "int", "int8", "int16", "int32", "int64",
            "uint", "uint8", "uint16", "uint32", "uint64", "uintptr",
            "float32", "float64", "complex64", "complex128", "any"
        ],
        lineCommentTokens: ["//"],
        stringDelimiters: [doubleQuote, backtick]
    )

    static let rust = Language(
        keywords: [
            "as", "break", "const", "continue", "crate", "else", "enum", "extern", "false", "fn",
            "for", "if", "impl", "in", "let", "loop", "match", "mod", "move", "mut", "pub", "ref",
            "return", "self", "Self", "static", "struct", "super", "trait", "true", "type",
            "unsafe", "use", "where", "while",
            "async", "await", "dyn"
        ],
        types: [
            "bool", "char", "str", "String", "Vec", "Option", "Result", "Box", "Rc", "Arc",
            "i8", "i16", "i32", "i64", "i128", "isize",
            "u8", "u16", "u32", "u64", "u128", "usize",
            "f32", "f64"
        ],
        lineCommentTokens: ["//"],
        stringDelimiters: [doubleQuote]
    )

    static let bash = Language(
        keywords: [
            "if", "then", "else", "elif", "fi", "case", "esac", "for", "while", "until", "do",
            "done", "in", "function", "return", "break", "continue", "exit", "echo", "printf",
            "export", "local", "readonly", "declare", "unset", "shift", "set", "source"
        ],
        types: [],
        lineCommentTokens: ["#"],
        stringDelimiters: [doubleQuote, singleQuote]
    )

    static let json = Language(
        keywords: ["true", "false", "null"],
        types: [],
        lineCommentTokens: [],
        stringDelimiters: [doubleQuote]
    )

    static let yaml = Language(
        keywords: ["true", "false", "null", "yes", "no", "on", "off"],
        types: [],
        lineCommentTokens: ["#"],
        stringDelimiters: [doubleQuote, singleQuote]
    )
}
