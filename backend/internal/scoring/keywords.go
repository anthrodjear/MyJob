// Package scoring provides job-candidate matching and scoring functionality.
// It supports three scoring modes: heuristic (keyword-based), LLM (semantic), and hybrid (pre-filter + LLM).
// The service computes factor scores (skills, experience, location, salary, description) and combines them
// into a final 0-100 score with approval tier (auto/review/reject).
package scoring

import (
	"strings"
)

// stopWords are common words that appear in job descriptions but aren't skills.
// These are filtered out during keyword extraction to reduce noise.
var stopWords = map[string]struct{}{
	"experience": {},
	"required":   {},
	"must":       {},
	"have":       {},
	"years":      {},
	"ability":    {},
	"working":    {},
	"knowledge":  {},
	"strong":     {},
	"excellent":  {},
	"good":       {},
	"least":      {},
	"minimum":    {},
	"including":  {},
	"using":      {},
	"work":       {},
	"team":       {},
	"role":       {},
	"position":   {},
	"job":        {},
	"company":    {},
	"looking":    {},
	"suitable":   {},
	"preferred":  {},
	"plus":       {},
	"benefits":   {},
	"salary":     {},
	"full":       {},
	"time":       {},
	"part":       {},
	"apply":      {},
	"please":     {},
	"send":       {},
	"resume":     {},
	"cv":         {},
	"email":      {},
	"click":      {},
}

// extractKeywords splits text into meaningful keywords, filtering stop-words.
func extractKeywords(text string) []string {
	words := strings.Fields(strings.ToLower(text))
	seen := make(map[string]struct{})
	var keywords []string
	for _, w := range words {
		w = strings.Trim(w, ".,;:!?()[]{}\"'")
		if len(w) < 3 {
			continue
		}
		if _, exists := stopWords[w]; exists {
			continue
		}
		if _, exists := seen[w]; exists {
			continue
		}
		seen[w] = struct{}{}
		keywords = append(keywords, w)
	}
	return keywords
}
