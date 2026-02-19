package envresolve

import (
	"fmt"
	"regexp"
	"strings"
)

// templatePattern matches ${{ ... }} references, capturing the inner content
// with optional surrounding whitespace trimmed.
var templatePattern = regexp.MustCompile(`\$\{\{\s*([^}]+?)\s*\}\}`)

// AppVar is an environment variable belonging to an app.
type AppVar struct {
	Key   string
	Value string
}

// SiblingVar is an environment variable belonging to a sibling app.
type SiblingVar struct {
	AppSlug string
	Key     string
	Value   string
}

// Resolve processes template references in app environment variables.
// It replaces ${{ ref }} patterns with actual values from the provided
// lookup maps.
//
// appVars: this app's own variables (for self-reference)
// sharedVars: environment-level shared variables (for ${{ shared.* }})
// siblingVars: variables from sibling apps (for ${{ app-slug.* }})
//
// Returns a map of key -> resolved value, or error if a referenced variable doesn't exist.
func Resolve(appVars []AppVar, sharedVars []AppVar, siblingVars []SiblingVar) (map[string]string, error) {
	// Build lookup maps.
	selfLookup := make(map[string]string, len(appVars))
	for _, v := range appVars {
		selfLookup[v.Key] = v.Value
	}

	sharedLookup := make(map[string]string, len(sharedVars))
	for _, v := range sharedVars {
		sharedLookup[v.Key] = v.Value
	}

	// siblingLookup is keyed by "appSlug.key".
	siblingLookup := make(map[string]string, len(siblingVars))
	for _, v := range siblingVars {
		siblingLookup[v.AppSlug+"."+v.Key] = v.Value
	}

	result := make(map[string]string, len(appVars))

	for _, av := range appVars {
		resolved, err := resolveValue(av.Value, selfLookup, sharedLookup, siblingLookup)
		if err != nil {
			return nil, fmt.Errorf("resolving variable %q: %w", av.Key, err)
		}
		result[av.Key] = resolved
	}

	return result, nil
}

// resolveValue replaces all ${{ ref }} patterns in a single value string.
func resolveValue(
	value string,
	selfLookup map[string]string,
	sharedLookup map[string]string,
	siblingLookup map[string]string,
) (string, error) {
	var resolveErr error

	resolved := templatePattern.ReplaceAllStringFunc(value, func(match string) string {
		if resolveErr != nil {
			return match
		}

		// Extract the captured group from the match.
		submatches := templatePattern.FindStringSubmatch(match)
		if len(submatches) < 2 {
			resolveErr = fmt.Errorf("invalid template syntax: %s", match)
			return match
		}

		ref := submatches[1]

		if dotIdx := strings.IndexByte(ref, '.'); dotIdx != -1 {
			scope := ref[:dotIdx]
			key := ref[dotIdx+1:]

			if scope == "shared" {
				val, ok := sharedLookup[key]
				if !ok {
					resolveErr = fmt.Errorf("shared variable %q not found", key)
					return match
				}
				return val
			}

			// Scope is an app slug; look up in sibling vars.
			lookupKey := scope + "." + key
			val, ok := siblingLookup[lookupKey]
			if !ok {
				resolveErr = fmt.Errorf("sibling app variable %q.%q not found", scope, key)
				return match
			}
			return val
		}

		// No dot: self-reference.
		val, ok := selfLookup[ref]
		if !ok {
			resolveErr = fmt.Errorf("self-referenced variable %q not found", ref)
			return match
		}
		return val
	})

	if resolveErr != nil {
		return "", resolveErr
	}

	return resolved, nil
}
