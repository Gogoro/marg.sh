import Foundation

struct FuzzyMatcher {
    static func match(query: String, candidates: [String]) -> [(index: Int, score: Int)] {
        let trimmed = query.trimmingCharacters(in: .whitespaces).lowercased()
        if trimmed.isEmpty {
            return candidates.indices.map { ($0, 0) }
        }

        var matches: [(index: Int, score: Int)] = []
        for (i, candidate) in candidates.enumerated() {
            let lowered = candidate.lowercased()
            if let score = scoreSubsequence(query: trimmed, in: lowered) {
                matches.append((i, score))
            }
        }
        matches.sort { lhs, rhs in
            if lhs.score != rhs.score {
                return lhs.score > rhs.score
            }
            return candidates[lhs.index].lowercased() < candidates[rhs.index].lowercased()
        }
        return matches
    }

    private static func scoreSubsequence(query: String, in candidate: String) -> Int? {
        let queryChars = Array(query)
        let candidateChars = Array(candidate)
        guard !queryChars.isEmpty else { return 0 }

        var qi = 0
        var lastMatchIndex: Int? = nil
        var consecutive = 0
        var score = 0

        for (ci, char) in candidateChars.enumerated() {
            if qi < queryChars.count && char == queryChars[qi] {
                score += 10
                if let last = lastMatchIndex, last + 1 == ci {
                    consecutive += 1
                    score += consecutive * 5
                } else {
                    consecutive = 0
                }
                if ci == 0 || candidateChars[ci - 1] == "/" {
                    score += 15
                }
                if ci == 0 {
                    score += 10
                }
                lastMatchIndex = ci
                qi += 1
            }
        }
        guard qi == queryChars.count else { return nil }
        score -= candidateChars.count
        return score
    }
}
