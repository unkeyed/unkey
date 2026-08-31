package main

import "bytes"

func splitTopLevel(text []byte, separator byte) [][]byte {
	var parts [][]byte
	start := 0
	depth := 0
	var quote byte
	for i := 0; i < len(text); i++ {
		char := text[i]
		if quote != 0 {
			if char == quote {
				quote = 0
			}
			continue
		}
		switch char {
		case '\'', '"', '`':
			quote = char
		case '(', '[', '{':
			depth++
		case ')', ']', '}':
			if depth > 0 {
				depth--
			}
		default:
			if char == separator && depth == 0 {
				parts = append(parts, text[start:i])
				start = i + 1
			}
		}
	}
	return append(parts, text[start:])
}

func matchingDelimiter(text []byte, start int) int {
	if start >= len(text) {
		return -1
	}
	first, ok := closingDelimiter(text[start])
	if !ok {
		return -1
	}
	stack := []byte{first}
	var quote byte
	for i := start + 1; i < len(text); i++ {
		char := text[i]
		if quote != 0 {
			if char == quote {
				quote = 0
			}
			continue
		}
		if isQuote(char) {
			quote = char
			continue
		}
		if close, found := closingDelimiter(char); found {
			stack = append(stack, close)
			continue
		}
		if char == stack[len(stack)-1] {
			stack = stack[:len(stack)-1]
			if len(stack) == 0 {
				return i
			}
		}
	}
	return -1
}

func closingDelimiter(open byte) (byte, bool) {
	switch open {
	case '(':
		return ')', true
	case '[':
		return ']', true
	case '{':
		return '}', true
	default:
		return 0, false
	}
}

func maskSQL(source []byte) []byte {
	masked := bytes.Clone(source)
	for i := 0; i < len(masked); {
		switch {
		case i+1 < len(masked) && masked[i] == '-' && masked[i+1] == '-':
			end := bytes.IndexByte(masked[i:], '\n')
			if end < 0 {
				end = len(masked) - i
			}
			blankInPlace(masked[i : i+end])
			i += end
		case i+1 < len(masked) && masked[i] == '/' && masked[i+1] == '*':
			end := bytes.Index(masked[i+2:], []byte("*/"))
			if end < 0 {
				end = len(masked) - i - 2
			}
			length := end + 4
			blankInPlace(masked[i:min(i+length, len(masked))])
			i += length
		case masked[i] == '\'':
			start := i
			i++
			for i < len(masked) {
				if masked[i] != '\'' {
					i++
					continue
				}
				if i+1 < len(masked) && masked[i+1] == '\'' {
					i += 2
					continue
				}
				i++
				break
			}
			blankInPlace(masked[start:i])
		default:
			i++
		}
	}
	return masked
}

func maskSourceComments(source []byte, goSource bool) []byte {
	masked := bytes.Clone(source)
	var quote byte
	for i := 0; i < len(masked); i++ {
		char := masked[i]
		if quote != 0 {
			if char == '\\' && (!goSource || quote != '`') && i+1 < len(masked) {
				i++
				continue
			}
			if char == quote {
				quote = 0
			}
			continue
		}
		if isQuote(char) {
			quote = char
			continue
		}
		if i+1 < len(masked) && char == '/' && masked[i+1] == '/' {
			end := bytes.IndexByte(masked[i:], '\n')
			if end < 0 {
				end = len(masked) - i
			}
			blankInPlace(masked[i : i+end])
			i += end - 1
			continue
		}
		if i+1 < len(masked) && char == '/' && masked[i+1] == '*' {
			end := bytes.Index(masked[i+2:], []byte("*/"))
			if end < 0 {
				end = len(masked) - i - 2
			}
			length := end + 4
			blankInPlace(masked[i:min(i+length, len(masked))])
			i += length - 1
		}
	}
	return masked
}

func maskTypeScriptStructure(source []byte) []byte {
	masked := maskSourceComments(source, false)
	var quote byte
	for i := 0; i < len(masked); i++ {
		char := masked[i]
		if quote == 0 {
			if isQuote(char) {
				quote = char
			}
			continue
		}
		if char == '\\' && i+1 < len(masked) {
			masked[i] = ' '
			i++
			if masked[i] != '\n' {
				masked[i] = ' '
			}
			continue
		}
		if char == quote {
			quote = 0
			continue
		}
		if char != '\n' {
			masked[i] = ' '
		}
	}
	return masked
}

func blank(match []byte) []byte {
	result := bytes.Clone(match)
	blankInPlace(result)
	return result
}

func blankInPlace(text []byte) {
	for i := range text {
		if text[i] != '\n' {
			text[i] = ' '
		}
	}
}

func skipSpace(text []byte, position int) int {
	for position < len(text) && isSpace(text[position]) {
		position++
	}
	return position
}

func trimSpaceEnd(text []byte, start, end int) int {
	for end > start && isSpace(text[end-1]) {
		end--
	}
	return end
}

func isSpace(char byte) bool {
	return char == ' ' || char == '\t' || char == '\n' || char == '\r' || char == '\f'
}

func isQuote(char byte) bool {
	return char == '\'' || char == '"' || char == '`'
}

func isWordStart(char byte) bool {
	return char == '_' || char >= 'A' && char <= 'Z' || char >= 'a' && char <= 'z'
}

func isWordPart(char byte) bool {
	return isWordStart(char) || char >= '0' && char <= '9'
}
