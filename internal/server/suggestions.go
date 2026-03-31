package server

import (
	"sort"
	"strings"

	"github.com/larsartmann/dynamic-markdown-site/internal/domain"
)

// minScoreThreshold is the minimum score for a suggestion to be considered relevant.
const minScoreThreshold = 0.3

// SuggestedPath represents a path suggestion with similarity score.
type SuggestedPath struct {
	Path  domain.URLPath
	Title string
	Score float64
}

// findSuggestions returns similar paths based on Levenshtein distance.
// It returns up to maxSuggestions paths, sorted by similarity (highest first).
// Only suggestions with score >= minScoreThreshold are returned.
func findSuggestions(requested string, paths []domain.URLPath, maxSuggestions int) []SuggestedPath {
	if len(paths) == 0 || requested == "" {
		return nil
	}

	requestedLower := strings.ToLower(requested)
	var suggestions []SuggestedPath

	for _, path := range paths {
		pathStr := path.String()
		pathLower := strings.ToLower(pathStr)

		if pathLower == requestedLower {
			continue // Skip exact matches (case-insensitive)
		}

		distance := levenshteinDistance(requestedLower, pathLower)
		maxLen := max(len(requested), len(pathStr))

		if maxLen == 0 {
			continue
		}

		// Normalize score: 1.0 = identical, 0.0 = completely different
		score := 1.0 - float64(distance)/float64(maxLen)

		// Boost score for path prefix matches
		if strings.HasPrefix(pathLower, requestedLower) ||
			strings.HasPrefix(requestedLower, pathLower) {
			score += 0.2
		}

		// Boost score for substring matches
		if strings.Contains(pathLower, requestedLower) ||
			strings.Contains(requestedLower, pathLower) {
			score += 0.1
		}

		// Only include suggestions above threshold
		if score >= minScoreThreshold {
			suggestions = append(suggestions, SuggestedPath{
				Path:  path,
				Title: path.Filename(),
				Score: score,
			})
		}
	}

	// Sort by score descending (higher score = better match)
	sort.Slice(suggestions, func(i, j int) bool {
		return suggestions[i].Score > suggestions[j].Score
	})

	// Return top N suggestions
	if len(suggestions) > maxSuggestions {
		suggestions = suggestions[:maxSuggestions]
	}

	return suggestions
}

// levenshteinDistance computes the edit distance between two strings.
// This implementation uses O(min(n,m)) space.
func levenshteinDistance(a, b string) int {
	if len(a) == 0 {
		return len(b)
	}

	if len(b) == 0 {
		return len(a)
	}

	// Ensure a is the shorter string to minimize space
	if len(a) > len(b) {
		a, b = b, a
	}

	// Previous row of the matrix
	prev := make([]int, len(a)+1)
	// Current row of the matrix
	curr := make([]int, len(a)+1)

	// Initialize first row
	for i := range prev {
		prev[i] = i
	}

	// Compute each row
	for j := 1; j <= len(b); j++ {
		curr[0] = j

		for i := 1; i <= len(a); i++ {
			cost := 0
			if a[i-1] != b[j-1] {
				cost = 1
			}

			curr[i] = min(
				curr[i-1]+1,      // deletion
				prev[i]+1,        // insertion
				prev[i-1]+cost,   // substitution
			)
		}

		// Swap rows
		prev, curr = curr, prev
	}

	return prev[len(a)]
}
