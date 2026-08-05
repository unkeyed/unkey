package validation

// Suggesting the property a caller probably meant.
//
// An unknown property is usually a typo, and the old message diagnosed it by
// quoting what was sent: `additional properties 'name:' not allowed`. That names
// the culprit exactly, and we cannot do it, because the property name comes from
// the request and this response is written to the request log. A caller posting
// `{"sk_live_abc": 1}` would have stored their own key.
//
// The way out is that the *answer* is spec data even though the question is not.
// `name:` is one edit away from `name`, which the schema declares, so naming the
// declared property pinpoints the typo without repeating a single byte the caller
// sent. What was sent is read here and never leaves this file.

// maxSuggestionDistance is the edit distance at which two names are still
// plausibly the same name mistyped. Short names get a tighter bound, because at
// distance 2 nearly every three-letter word reaches every other one.
func maxSuggestionDistance(declared string) int {
	if len(declared) <= 4 {
		return 1
	}

	return 2
}

// nearestDeclared returns the declared property a caller most likely meant, given
// the names they actually sent. It reports false when nothing is close enough,
// when several declared names tie, or when more than one sent name has a match,
// since a suggestion is only useful while it is unambiguous.
func nearestDeclared(sent, declared []string) (string, bool) {
	best := ""
	bestDistance := -1
	ambiguous := false

	for _, unknown := range sent {
		for _, candidate := range declared {
			distance := editDistance(unknown, candidate)
			if distance == 0 || distance > maxSuggestionDistance(candidate) {
				continue
			}

			switch {
			case bestDistance < 0 || distance < bestDistance:
				best, bestDistance, ambiguous = candidate, distance, false
			case distance == bestDistance && candidate != best:
				ambiguous = true
			}
		}
	}

	if bestDistance < 0 || ambiguous {
		return "", false
	}

	return specToken(best)
}

// editDistance is Levenshtein distance, over bytes. Property names are short, and
// a multi-byte character counted as several edits only ever makes a suggestion
// less likely, never wrong.
func editDistance(a, b string) int {
	if a == b {
		return 0
	}
	if len(a) == 0 {
		return len(b)
	}
	if len(b) == 0 {
		return len(a)
	}

	// Two rows are enough: each cell depends only on the row above and the cell
	// to its left.
	previous := make([]int, len(b)+1)
	current := make([]int, len(b)+1)
	for j := range previous {
		previous[j] = j
	}

	for i := 1; i <= len(a); i++ {
		current[0] = i
		for j := 1; j <= len(b); j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			current[j] = min(min(current[j-1]+1, previous[j]+1), previous[j-1]+cost)
		}
		previous, current = current, previous
	}

	return previous[len(b)]
}
