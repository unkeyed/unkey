package main

import (
	"bytes"
	"regexp"
	"strings"
)

var (
	sqlcEmbedPattern     = regexp.MustCompile(`(?i)\bsqlc\s*\.\s*embed\s*\(`)
	selectPattern        = regexp.MustCompile(`(?i)\bselect\b`)
	qualifiedStarPattern = regexp.MustCompile("(?i)(?:`[^`]+`|[A-Za-z_][A-Za-z0-9_]*)\\s*\\.\\s*\\*")

	// Applied Atlas migrations are immutable. Forward migrations replace these
	// views, so only the exact historical statements are masked.
	migrationMasks = map[string][]*regexp.Regexp{
		"pkg/clickhouse/migrations/20260419000000.sql": {
			regexp.MustCompile(`(?i)SELECT \* FROM default\.instance_checkpoints_v1 FINAL`),
		},
		"pkg/clickhouse/migrations/20260804000000.sql": {
			regexp.MustCompile(`(?is)SELECT \*\s+FROM default\.instance_checkpoints_v1\s+FINAL`),
		},
	}
)

func (r *reporter) reportSQLWildcards(path string, original, source []byte) {
	for _, match := range sqlcEmbedPattern.FindAllIndex(source, -1) {
		r.report(path, original, match[0], "sqlc.embed all-column selection")
	}

	for _, match := range selectPattern.FindAllIndex(source, -1) {
		projection, ok := selectProjection(source, match[1])
		if !ok {
			continue
		}
		if hasBareWildcard(projection) {
			r.report(path, original, match[0], "SELECT wildcard")
		}
		if qualifiedStarPattern.Match(projection) {
			r.report(path, original, match[0], "qualified wildcard selection")
		}
	}
}

func selectProjection(source []byte, start int) ([]byte, bool) {
	depth := 0
	var quote byte
	for i := start; i < len(source); i++ {
		char := source[i]
		if quote != 0 {
			if char == '\\' && quote != '`' && i+1 < len(source) {
				i++
				continue
			}
			if char == quote {
				quote = 0
			}
			continue
		}
		switch char {
		case '\'', '"', '`':
			quote = char
		case '(':
			depth++
		case ')':
			if depth > 0 {
				depth--
			}
		case ';':
			if depth == 0 {
				return nil, false
			}
		default:
			if depth == 0 && isWordStart(char) {
				end := i + 1
				for end < len(source) && isWordPart(source[end]) {
					end++
				}
				if strings.EqualFold(string(source[i:end]), "from") {
					return source[start:i], true
				}
				i = end - 1
			}
		}
	}
	return nil, false
}

func hasBareWildcard(projection []byte) bool {
	segments := splitTopLevel(projection, ',')
	for i, segment := range segments {
		segment = bytes.TrimSpace(segment)
		if i == 0 {
			segment = trimSelectModifiers(segment)
		}
		if len(segment) > 0 && segment[0] == '*' && (len(segment) == 1 || isSpace(segment[1])) {
			return true
		}
	}
	return false
}

func trimSelectModifiers(segment []byte) []byte {
	for {
		segment = bytes.TrimSpace(segment)
		end := 0
		for end < len(segment) && isWordPart(segment[end]) {
			end++
		}
		if end == 0 || !isSelectModifier(string(segment[:end])) {
			return segment
		}
		segment = segment[end:]
	}
}

func isSelectModifier(word string) bool {
	switch strings.ToLower(word) {
	case "all", "distinct", "distinctrow", "high_priority", "straight_join",
		"sql_small_result", "sql_big_result", "sql_buffer_result", "sql_no_cache",
		"sql_calc_found_rows":
		return true
	default:
		return false
	}
}
