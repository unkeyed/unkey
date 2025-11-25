package rbac

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/unkeyed/unkey/go/pkg/codes"
	"github.com/unkeyed/unkey/go/pkg/fault"
)

// tokenType represents the different types of tokens that can be found in a
// SQL-like permission query string during lexical analysis.
//
// The lexer recognizes six distinct token types: permission identifiers,
// logical operators (AND, OR), parentheses for grouping, end-of-input marker,
// and error tokens for invalid characters.
type tokenType int

const (
	// permission represents a permission string token like "api.key1.read_key".
	// Permission tokens can contain alphanumeric characters, dots, underscores,
	// and hyphens. They are the leaf nodes in the permission query AST.
	permission tokenType = iota

	// and represents the logical AND operator token.
	// Recognized in any case variation: "AND", "and", "And", etc.
	// AND has higher precedence than OR in the parser.
	and

	// or represents the logical OR operator token.
	// Recognized in any case variation: "OR", "or", "Or", etc.
	// OR has lower precedence than AND in the parser.
	or

	// lparen represents a left parenthesis "(" token.
	// Used for grouping expressions and overriding default operator precedence.
	lparen

	// rparen represents a right parenthesis ")" token.
	// Must be properly matched with lparen tokens.
	rparen

	// eof represents the end-of-input token.
	// Signals that the lexer has reached the end of the input string.
	eof

	// errorToken represents an invalid token caused by illegal characters.
	// The token's value field contains the error message.
	errorToken
)

// token represents a single lexical token extracted from a permission query string.
//
// Each token contains its type classification, the original string value from the
// input, and its position for error reporting. Position tracking is essential for
// providing helpful error messages to users when parsing fails.
type token struct {
	// typ classifies the token into one of the predefined token types
	typ tokenType

	// value contains the original string text that produced this token
	value string

	// pos indicates the starting position of this token in the original input string.
	// Used for generating precise error messages with location information.
	pos int
}

// lexer performs lexical analysis on permission query strings, breaking them
// into tokens that can be consumed by the parser.
//
// The lexer implements a single-pass tokenizer that recognizes permission
// identifiers, logical operators (AND, OR), parentheses, and whitespace.
// It maintains position tracking for error reporting and handles case-insensitive
// operator recognition.
//
// The lexer is designed to be efficient for typical permission query sizes
// (under 1000 characters) and provides detailed error messages for invalid
// input characters.
type lexer struct {
	// input contains the complete permission query string being tokenized
	input string

	// pos tracks the current position in the input string.
	// Points to the next character to be read.
	pos int

	// ch contains the current character being examined.
	// Set to 0 (ASCII NUL) when end of input is reached.
	ch byte
}

// newLexer creates a new lexer instance for tokenizing the given permission query string.
//
// The lexer is initialized with the input string and positioned at the first character.
// It immediately reads the first character to prepare for token generation.
//
// The input string can contain any valid permission query syntax including permissions,
// logical operators, parentheses, and whitespace. Invalid characters will be detected
// during tokenization and result in error tokens.
//
// Example usage:
//
//	lexer := newLexer("api.key1.read_key AND api.key1.write_key")
//	for {
//	    token := lexer.nextToken()
//	    if token.typ == eof {
//	        break
//	    }
//	    // Process token...
//	}
func newLexer(input string) *lexer {
	l := &lexer{
		ch:    0,
		input: input,
		pos:   0,
	}
	l.readChar()
	return l
}

// readChar advances the lexer to the next character in the input string.
//
// When the end of input is reached, ch is set to 0 (ASCII NUL) to signal EOF.
// The position is always advanced, even at EOF, to maintain consistent state.
//
// This method is called internally by other lexer methods and should not be
// called directly by external code.
func (l *lexer) readChar() {
	if l.pos >= len(l.input) {
		l.ch = 0 // ASCII NUL character represents EOF
	} else {
		l.ch = l.input[l.pos]
	}
	l.pos++
}

// skipWhitespace advances the lexer past any whitespace characters.
//
// Recognizes and skips spaces, tabs, newlines, and carriage returns.
// This allows permission queries to be formatted with arbitrary whitespace
// for readability without affecting the parsing logic.
//
// Whitespace is not significant in permission queries and is discarded
// during tokenization. The lexer will skip any amount of consecutive
// whitespace characters.
func (l *lexer) skipWhitespace() {
	for l.ch == ' ' || l.ch == '\t' || l.ch == '\n' || l.ch == '\r' {
		l.readChar()
	}
}

// readIdentifier reads a complete identifier token from the input.
//
// An identifier can be either a permission string or an operator keyword.
// It continues reading characters until it encounters a character that
// is not valid in a permission identifier.
//
// Valid permission characters are defined by [isValidPermissionChar] and
// include alphanumeric characters, dots, underscores, and hyphens.
//
// The method returns the complete identifier string. The caller is responsible
// for determining whether the identifier is a permission or an operator keyword.
func (l *lexer) readIdentifier() string {
	position := l.pos - 1 // Start from current character

	// Read all valid identifier characters
	for isValidPermissionChar(l.ch) {
		l.readChar()
	}

	return l.input[position : l.pos-1]
}

// isValidPermissionChar determines whether a character is allowed in permission identifiers.
//
// Permission identifiers can contain:
//   - Alphanumeric characters (letters and digits)
//   - Dots (.) for hierarchical separation (e.g., "api.key1.read_key")
//   - Underscores (_) for word separation
//   - Hyphens (-) for kebab-case identifiers
//   - Colons (:) for namespace separation (e.g., "system:admin")
//   - Asterisks (*) for literal permission names (e.g., "api.*")
//   - Forward slashes (/) for path-like permission names (e.g., "/api/v1/xxx")
//
// Note: The asterisk (*) character is treated as a literal character in permission
// names, NOT as a wildcard pattern. For example, "api.*" matches only the exact
// permission "api.*", not "api.read" or "api.write".
//
// This character set matches the regex: /^[a-zA-Z0-9_:\-\.\*\/]+$/
//
// Characters like spaces, parentheses, and operators are not allowed in
// permission identifiers and will terminate identifier parsing.
func isValidPermissionChar(ch byte) bool {
	return unicode.IsLetter(rune(ch)) || unicode.IsDigit(rune(ch)) || ch == '.' || ch == '_' || ch == '-' || ch == '*' || ch == ':' || ch == '/'
}

// nextToken extracts and returns the next token from the input stream.
//
// This is the main entry point for token generation. It skips whitespace,
// identifies the type of the next token, and returns a properly constructed
// token with type, value, and position information.
//
// Token recognition logic:
//   - Parentheses are recognized as single-character tokens
//   - Identifiers (starting with valid permission characters) are read completely
//   - Operators "AND" and "OR" are recognized case-insensitively but only as complete words
//   - End of input produces an EOF token
//   - Invalid characters produce error tokens with descriptive messages
//
// The lexer maintains position information for all tokens to enable precise
// error reporting in the parser.
//
// Returns a token struct containing the token type, original string value,
// and position in the input. Error tokens contain error messages in the value field.
func (l *lexer) nextToken() token {
	var tok token

	l.skipWhitespace()

	switch l.ch {
	case '(':
		tok = token{typ: lparen, value: string(l.ch), pos: l.pos - 1}
		l.readChar()
	case ')':
		tok = token{typ: rparen, value: string(l.ch), pos: l.pos - 1}
		l.readChar()
	case 0:
		tok = token{typ: eof, value: "", pos: l.pos - 1}
	default:
		if isValidPermissionChar(l.ch) {
			pos := l.pos - 1
			identifier := l.readIdentifier()

			// Check if it's an operator (case-insensitive) AND it's a complete word
			upperIdent := strings.ToUpper(identifier)
			if upperIdent == "AND" && l.isCompleteWord(pos, identifier) {
				tok = token{typ: and, value: identifier, pos: pos}
			} else if upperIdent == "OR" && l.isCompleteWord(pos, identifier) {
				tok = token{typ: or, value: identifier, pos: pos}
			} else {
				// It's a permission
				tok = token{typ: permission, value: identifier, pos: pos}
			}
			return tok
		} else {
			// Invalid character - create a structured error
			pos := l.pos - 1
			char := l.ch

			// Create an error with both internal debugging and user-friendly message
			err := fault.New(
				fmt.Sprintf("invalid character '%c' at position %d in query", char, pos),
				fault.Code(codes.User.BadRequest.PermissionsQuerySyntaxError.URN()),
				fault.Public(fmt.Sprintf("Invalid character '%c' found in query at position %d. Only letters, numbers, dots, underscores, hyphens, slashes, colons, asterisks, and parentheses are allowed.", char, pos)),
			)

			tok = token{
				typ:   errorToken,
				value: err.Error(),
				pos:   pos,
			}
			l.readChar()
		}
	}

	return tok
}

// isCompleteWord determines whether an identifier stands alone as a complete word.
//
// This check is essential for distinguishing between operator keywords ("AND", "OR")
// and permission identifiers that happen to contain these keywords ("ANDROID", "ORACLE").
//
// The method examines the characters immediately before and after the identifier
// to ensure it's not part of a larger permission identifier. For example:
//   - "AND" in "perm1 AND perm2" is a complete word (operator)
//   - "AND" in "ANDROID.permission" is not a complete word (part of permission)
//
// Returns true if the identifier is surrounded by non-identifier characters
// (whitespace, parentheses, or boundaries), false if it's part of a larger identifier.
func (l *lexer) isCompleteWord(startPos int, identifier string) bool {
	// Check character before (if exists)
	if startPos > 0 {
		prevChar := l.input[startPos-1]
		if isValidPermissionChar(prevChar) {
			return false
		}
	}

	// Check character after (if exists)
	endPos := startPos + len(identifier)
	if endPos < len(l.input) {
		nextChar := l.input[endPos]
		if isValidPermissionChar(nextChar) {
			return false
		}
	}

	return true
}
