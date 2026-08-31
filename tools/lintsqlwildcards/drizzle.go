package main

import (
	"bytes"
	"regexp"
	"strings"
)

var (
	drizzleQueryPattern = regexp.MustCompile(`\.\s*query\s*\.\s*[A-Za-z_$][A-Za-z0-9_$]*\s*\.\s*find(?:First|Many)\s*`)
	selectAllPattern    = regexp.MustCompile(`(?i)\.\s*selectAll\s*\(`)
	emptySelectPattern  = regexp.MustCompile(`(?i)\.\s*select(?:Distinct)?\s*\(\s*\)\s*\.\s*from\s*\(`)
	identifierPattern   = regexp.MustCompile(`^[A-Za-z_$][A-Za-z0-9_$]*$`)
)

type property struct {
	key        string
	keyValid   bool
	valueStart int
	valueEnd   int
	offset     int
}

func (r *reporter) reportDrizzleQueries(path string, original []byte) {
	masked := maskTypeScriptStructure(original)
	searchStart := 0
	for searchStart < len(masked) {
		match := drizzleQueryPattern.FindIndex(masked[searchStart:])
		if match == nil {
			return
		}
		callOffset := searchStart + match[0]
		position := skipSpace(masked, searchStart+match[1])
		if position >= len(masked) {
			return
		}
		if masked[position] == '<' {
			if genericCall(masked, position) {
				r.report(path, original, callOffset, "generic Drizzle relational query")
			}
			searchStart = position + 1
			continue
		}
		if masked[position] != '(' {
			searchStart = position + 1
			continue
		}

		close := matchingDelimiter(masked, position)
		if close < 0 {
			r.report(path, original, callOffset, "unbalanced Drizzle relational query")
			searchStart = position + 1
			continue
		}
		configStart := skipSpace(masked, position+1)
		configEnd := trimSpaceEnd(masked, configStart, close)
		r.validateConfig(path, original, masked, configStart, configEnd, "query")
		searchStart = close + 1
	}
}

func genericCall(source []byte, start int) bool {
	depth := 0
	for position := start; position < len(source); position++ {
		switch source[position] {
		case '<':
			depth++
		case '>':
			depth--
			if depth == 0 {
				position = skipSpace(source, position+1)
				return position < len(source) && source[position] == '('
			}
		case ';':
			return false
		}
	}
	return false
}

func (r *reporter) reportQueryBuilders(path string, original, source []byte) {
	for _, match := range selectAllPattern.FindAllIndex(source, -1) {
		r.report(path, original, match[0], "query-builder selectAll selection")
	}
	for _, match := range emptySelectPattern.FindAllIndex(source, -1) {
		r.report(path, original, match[0], "empty query-builder .select()")
	}

	lower := bytes.ToLower(source)
	for searchStart := 0; searchStart < len(source); {
		relative := bytes.Index(lower[searchStart:], []byte(".select"))
		if relative < 0 {
			return
		}
		offset := searchStart + relative
		position := skipSpace(source, offset+len(".select"))
		if position < len(source) && source[position] == '(' && selectArgumentIsWildcard(source, position+1) {
			r.report(path, original, offset, "query-builder string wildcard selection")
		}
		searchStart = offset + len(".select")
	}
}

func selectArgumentIsWildcard(source []byte, start int) bool {
	position := skipSpace(source, start)
	if position < len(source) && source[position] == '[' {
		position = skipSpace(source, position+1)
	}
	if position >= len(source) || !isQuote(source[position]) {
		return false
	}
	quote := source[position]
	position = skipSpace(source, position+1)
	if position >= len(source) {
		return false
	}

	star := bytes.IndexByte(source[position:], '*')
	if star < 0 {
		return false
	}
	star += position
	prefix := bytes.TrimSpace(source[position:star])
	if len(prefix) > 0 {
		if prefix[len(prefix)-1] != '.' {
			return false
		}
		name := bytes.TrimSpace(prefix[:len(prefix)-1])
		if !(identifierPattern.Match(name) || (len(name) >= 2 && name[0] == '`' && name[len(name)-1] == '`')) {
			return false
		}
	}
	position = skipSpace(source, star+1)
	return position < len(source) && source[position] == quote
}

func (r *reporter) validateConfig(path string, original, masked []byte, start, end int, context string) {
	open, close, ok := literalObjectBounds(masked, start, end)
	if !ok {
		r.report(path, original, start, "nonliteral Drizzle "+context+" config")
		return
	}
	foundColumns := false
	for _, prop := range objectProperties(masked, original, open, close) {
		if !prop.keyValid {
			continue
		}
		switch prop.key {
		case "columns":
			foundColumns = true
			r.validateColumns(path, original, masked, prop.valueStart, prop.valueEnd)
		case "with":
			r.validateWith(path, original, masked, prop.valueStart, prop.valueEnd)
		}
	}
	if !foundColumns {
		r.report(path, original, open, "missing Drizzle "+context+" columns projection")
	}
}

func (r *reporter) validateColumns(path string, original, masked []byte, start, end int) {
	open, close, ok := literalObjectBounds(masked, start, end)
	if !ok {
		r.report(path, original, start, "dynamic Drizzle columns projection")
		return
	}
	properties := objectProperties(masked, original, open, close)
	if len(properties) == 0 {
		r.report(path, original, open, "empty Drizzle columns projection")
		return
	}
	for _, prop := range properties {
		value := masked[prop.valueStart:prop.valueEnd]
		if !prop.keyValid || !bytes.Equal(value, []byte("true")) {
			r.report(path, original, prop.offset, "non-positive Drizzle columns projection")
		}
	}
}

func (r *reporter) validateWith(path string, original, masked []byte, start, end int) {
	open, close, ok := literalObjectBounds(masked, start, end)
	if !ok {
		r.report(path, original, start, "dynamic Drizzle with configuration")
		return
	}
	for _, relation := range objectProperties(masked, original, open, close) {
		if !relation.keyValid {
			r.report(path, original, relation.offset, "dynamic Drizzle relation configuration")
			continue
		}
		value := masked[relation.valueStart:relation.valueEnd]
		switch string(value) {
		case "false":
			continue
		case "true":
			r.report(path, original, relation.offset, "implicit all-column Drizzle relation")
		default:
			r.validateConfig(path, original, masked, relation.valueStart, relation.valueEnd, "relation")
		}
	}
}

func literalObjectBounds(masked []byte, start, end int) (int, int, bool) {
	if start >= end || start >= len(masked) || masked[start] != '{' {
		return 0, 0, false
	}
	close := matchingDelimiter(masked, start)
	if close < 0 || close > end || skipSpace(masked, close+1) < end {
		return 0, 0, false
	}
	return start, close, true
}

func objectProperties(masked, original []byte, open, close int) []property {
	ranges := topLevelRanges(masked, open+1, close)
	properties := make([]property, 0, len(ranges))
	for _, current := range ranges {
		begin := skipSpace(masked, current[0])
		end := current[1]
		if begin >= end {
			continue
		}
		colon := topLevelColon(masked, begin, end)
		if colon < 0 {
			properties = append(properties, property{
				key:        "",
				keyValid:   false,
				valueStart: 0,
				valueEnd:   0,
				offset:     begin,
			})
			continue
		}
		key, valid := propertyKey(original[begin:colon])
		valueStart := skipSpace(masked, colon+1)
		valueEnd := trimSpaceEnd(masked, valueStart, end)
		properties = append(properties, property{
			key:        key,
			keyValid:   valid,
			valueStart: valueStart,
			valueEnd:   valueEnd,
			offset:     begin,
		})
	}
	return properties
}

func topLevelRanges(masked []byte, start, close int) [][2]int {
	var ranges [][2]int
	rangeStart := start
	depth := 0
	var quote byte
	for i := start; i <= close; i++ {
		char := byte(',')
		if i < close {
			char = masked[i]
		}
		if quote != 0 {
			if char == quote {
				quote = 0
			}
			continue
		}
		switch char {
		case '\'', '"', '`':
			quote = char
		case '(', '{', '[':
			depth++
		case ')', '}', ']':
			if depth > 0 {
				depth--
			}
		case ',':
			if depth == 0 {
				ranges = append(ranges, [2]int{rangeStart, i})
				rangeStart = i + 1
			}
		}
	}
	return ranges
}

func topLevelColon(masked []byte, start, end int) int {
	depth := 0
	var quote byte
	for i := start; i < end; i++ {
		char := masked[i]
		if quote != 0 {
			if char == quote {
				quote = 0
			}
			continue
		}
		switch char {
		case '\'', '"':
			quote = char
		case '(', '{', '[':
			depth++
		case ')', '}', ']':
			if depth > 0 {
				depth--
			}
		case ':':
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}

func propertyKey(raw []byte) (string, bool) {
	text := strings.TrimSpace(string(raw))
	if identifierPattern.MatchString(text) {
		return text, true
	}
	if len(text) >= 2 && (text[0] == '\'' || text[0] == '"') && text[len(text)-1] == text[0] {
		return text[1 : len(text)-1], true
	}
	return "", false
}
