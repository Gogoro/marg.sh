import Foundation

enum TableFormatter {
    static func format(_ text: String) -> String {
        let lines = text.components(separatedBy: "\n")
        var output: [String] = []
        var index = 0
        while index < lines.count {
            if let result = tryFormatTable(at: index, in: lines) {
                output.append(contentsOf: result.formattedLines)
                index += result.consumed
            } else {
                output.append(lines[index])
                index += 1
            }
        }
        return output.joined(separator: "\n")
    }

    private struct TableFormatResult {
        let formattedLines: [String]
        let consumed: Int
    }

    private enum Alignment {
        case left
        case right
        case center
        case none
    }

    private static func tryFormatTable(at start: Int, in lines: [String]) -> TableFormatResult? {
        guard start + 1 < lines.count else { return nil }
        guard isTableRow(lines[start]) else { return nil }
        guard isAlignmentRow(lines[start + 1]) else { return nil }

        var rawRows: [[String]] = []
        rawRows.append(splitCells(lines[start]))
        let alignmentSpecs = parseAlignmentRow(lines[start + 1])

        var cursor = start + 2
        while cursor < lines.count, isTableRow(lines[cursor]) {
            rawRows.append(splitCells(lines[cursor]))
            cursor += 1
        }
        let consumed = cursor - start

        let columnCount = max(alignmentSpecs.count, rawRows.map { $0.count }.max() ?? 0)
        guard columnCount > 0 else { return nil }

        let normalizedRows = rawRows.map { row -> [String] in
            var padded = row
            while padded.count < columnCount { padded.append("") }
            return padded
        }
        var normalizedAlignment = alignmentSpecs
        while normalizedAlignment.count < columnCount { normalizedAlignment.append(.none) }

        var widths: [Int] = Array(repeating: 0, count: columnCount)
        for row in normalizedRows {
            for (column, cell) in row.enumerated() {
                widths[column] = max(widths[column], cell.count)
            }
        }
        for column in 0..<columnCount {
            widths[column] = max(widths[column], 3)
        }

        var formattedLines: [String] = []
        formattedLines.append(formatRow(normalizedRows[0], widths: widths, alignment: normalizedAlignment))
        formattedLines.append(formatAlignmentRow(widths: widths, alignment: normalizedAlignment))
        for row in normalizedRows.dropFirst() {
            formattedLines.append(formatRow(row, widths: widths, alignment: normalizedAlignment))
        }
        return TableFormatResult(formattedLines: formattedLines, consumed: consumed)
    }

    private static func isTableRow(_ line: String) -> Bool {
        guard line.hasPrefix("|"), line.count >= 2 else { return false }
        var index = line.endIndex
        while index > line.startIndex {
            index = line.index(before: index)
            let character = line[index]
            if !character.isWhitespace {
                return character == "|"
            }
        }
        return false
    }

    private static func isAlignmentRow(_ line: String) -> Bool {
        guard isTableRow(line) else { return false }
        let cells = splitCells(line)
        guard !cells.isEmpty else { return false }
        for cell in cells {
            if !isAlignmentSpec(cell) { return false }
        }
        return true
    }

    private static func isAlignmentSpec(_ cell: String) -> Bool {
        var content = cell
        if content.first == ":" { content.removeFirst() }
        if content.last == ":" { content.removeLast() }
        guard !content.isEmpty else { return false }
        return content.allSatisfy { $0 == "-" }
    }

    private static func parseAlignmentRow(_ line: String) -> [Alignment] {
        return splitCells(line).map { cell -> Alignment in
            let leftColon = cell.first == ":"
            let rightColon = cell.last == ":"
            switch (leftColon, rightColon) {
            case (true, true): return .center
            case (false, true): return .right
            case (true, false): return .left
            case (false, false): return .none
            }
        }
    }

    private static func splitCells(_ line: String) -> [String] {
        let trimmed = line.trimmingCharacters(in: .whitespaces)
        guard trimmed.hasPrefix("|"), trimmed.hasSuffix("|"), trimmed.count >= 2 else { return [] }

        let innerStart = trimmed.index(after: trimmed.startIndex)
        let innerEnd = trimmed.index(before: trimmed.endIndex)
        let inner = String(trimmed[innerStart..<innerEnd])

        var cells: [String] = []
        var current = ""
        var index = inner.startIndex
        while index < inner.endIndex {
            let character = inner[index]
            let next = inner.index(after: index)
            if character == "\\", next < inner.endIndex, inner[next] == "|" {
                current.append("\\|")
                index = inner.index(after: next)
                continue
            }
            if character == "|" {
                cells.append(current.trimmingCharacters(in: .whitespaces))
                current = ""
            } else {
                current.append(character)
            }
            index = next
        }
        cells.append(current.trimmingCharacters(in: .whitespaces))
        return cells
    }

    private static func formatRow(_ row: [String], widths: [Int], alignment: [Alignment]) -> String {
        var parts: [String] = []
        for (column, cell) in row.enumerated() {
            let width = widths[column]
            parts.append(padCell(cell, width: width, alignment: alignment[column]))
        }
        return "| " + parts.joined(separator: " | ") + " |"
    }

    private static func padCell(_ cell: String, width: Int, alignment: Alignment) -> String {
        let pad = max(0, width - cell.count)
        switch alignment {
        case .right:
            return String(repeating: " ", count: pad) + cell
        case .center:
            let leftPad = pad / 2
            let rightPad = pad - leftPad
            return String(repeating: " ", count: leftPad) + cell + String(repeating: " ", count: rightPad)
        case .left, .none:
            return cell + String(repeating: " ", count: pad)
        }
    }

    private static func formatAlignmentRow(widths: [Int], alignment: [Alignment]) -> String {
        var parts: [String] = []
        for (column, spec) in alignment.enumerated() {
            let width = widths[column]
            switch spec {
            case .left:
                parts.append(":" + String(repeating: "-", count: width - 1))
            case .right:
                parts.append(String(repeating: "-", count: width - 1) + ":")
            case .center:
                parts.append(":" + String(repeating: "-", count: width - 2) + ":")
            case .none:
                parts.append(String(repeating: "-", count: width))
            }
        }
        return "| " + parts.joined(separator: " | ") + " |"
    }
}
