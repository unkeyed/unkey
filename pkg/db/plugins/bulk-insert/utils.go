package main

import (
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

var caser = cases.Title(language.English)

// pluralize converts a singular word to its plural form
func pluralize(word string) string {
	if word == "" {
		return word
	}

	lower := strings.ToLower(word)

	// Handle regular pluralization rules
	if strings.HasSuffix(lower, "s") || strings.HasSuffix(lower, "sh") ||
		strings.HasSuffix(lower, "ch") || strings.HasSuffix(lower, "x") ||
		strings.HasSuffix(lower, "z") {
		return word + "es"
	}

	if strings.HasSuffix(lower, "y") && len(word) > 1 {
		lastChar := word[len(word)-2]
		if lastChar != 'a' && lastChar != 'e' && lastChar != 'i' && lastChar != 'o' && lastChar != 'u' {
			return word[:len(word)-1] + "ies"
		}
	}

	// Default: just add 's'
	return word + "s"
}

func ToCamelCase(name string) string {
	out := ""

	for i, p := range strings.Split(name, "_") {
		if p == "id" {
			out += "ID"
		} else if p == "url" && i > 0 {
			// sqlc uses "Url" not "URL" in compound names
			out += "Url"
		} else {
			out += caser.String(p)
		}
	}

	return out
}
