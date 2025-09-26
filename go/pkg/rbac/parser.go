package rbac

import (
	"fmt"

	"github.com/unkeyed/unkey/go/pkg/codes"
	"github.com/unkeyed/unkey/go/pkg/fault"
)

// parser implements a recursive descent parser for SQL-like permission queries.
//
// The parser consumes tokens from a lexer and builds an Abstract Syntax Tree (AST)
// represented as a [PermissionQuery] structure. It implements proper operator precedence
// (AND has higher precedence than OR) and supports parenthetical grouping to override
// default precedence.
//
// The parser uses a two-token lookahead strategy to make parsing decisions and provide
// better error messages. It maintains currentToken for the token being processed and
// nextToken for lookahead.
//
// Grammar implemented:
//
//	expression    → andExpression (OR andExpression)*
//	andExpression → primary (AND primary)*
//	primary       → PERMISSION | LPAREN expression RPAREN
//
// This grammar ensures AND has higher precedence than OR, matching SQL conventions.
//
// Note on asterisk (*) characters:
// Asterisks are treated as literal characters in permission names, NOT as wildcard
// patterns. For example, "api.*" will only match a permission literally named "api.*",
// not permissions like "api.read" or "api.write". This allows for exact permission
// matching without pattern expansion.
type parser struct {
	// lexer provides the token stream for parsing
	lexer *lexer

	// currentToken is the token currently being processed
	currentToken token

	// nextToken provides one-token lookahead for parsing decisions
	nextToken token
}

// newParser creates a new parser instance for the given permission query string.
//
// The parser is initialized with a lexer for the input string and immediately
// reads two tokens to establish the current token and lookahead token state.
// This two-token lookahead is necessary for proper error reporting and parsing decisions.
//
// Example usage:
//
//	parser := newParser("api.key1.read_key AND (api.key2.read_key OR api.key3.read_key)")
//	query, err := parser.parse()
//	if err != nil {
//	    // Handle parsing error
//	}
//	// Use the resulting PermissionQuery...
func newParser(input string) *parser {
	p := &parser{
		lexer: newLexer(input),
	}

	// Read two tokens so currentToken and nextToken are both set
	p.nextToken = p.lexer.nextToken()
	p.readToken()

	return p
}

// readToken advances the parser by one token in the input stream.
//
// This method implements the token advancement mechanism for the parser's
// two-token lookahead system. It moves nextToken to currentToken and
// reads a new token from the lexer into nextToken.
//
// This method is called internally by parsing methods and should not be
// called directly by external code.
func (p *parser) readToken() {
	p.currentToken = p.nextToken
	p.nextToken = p.lexer.nextToken()
}

// parseExpression parses a complete expression with OR operators at the top level.
//
// This method handles the lowest precedence level in the grammar, parsing
// expressions of the form "andExpression OR andExpression OR ...". It implements
// left-associative parsing, so "A OR B OR C" is parsed as "(A OR B) OR C".
//
// The method first parses the left operand as an AND expression, then continues
// parsing additional OR clauses until no more OR operators are found.
//
// Returns the parsed [PermissionQuery] representing the complete OR expression,
// or an error if parsing fails at any point.
func (p *parser) parseExpression() (PermissionQuery, error) {
	left, err := p.parseAndExpression()
	if err != nil {
		return PermissionQuery{}, err
	}

	// Handle OR operators (lowest precedence)
	for p.currentToken.typ == or {
		p.readToken() // consume OR
		right, err := p.parseAndExpression()
		if err != nil {
			return PermissionQuery{}, err
		}
		left = Or(left, right)
	}

	return left, nil
}

// parseAndExpression parses expressions with AND operators at the middle precedence level.
//
// This method handles expressions of the form "primary AND primary AND ...", where
// primary can be either a permission identifier or a parenthesized expression.
// AND operators have higher precedence than OR operators, so "A OR B AND C" is
// parsed as "A OR (B AND C)".
//
// The method implements left-associative parsing, so "A AND B AND C" is parsed
// as "(A AND B) AND C".
//
// Returns the parsed [PermissionQuery] representing the complete AND expression,
// or an error if parsing fails at any point.
func (p *parser) parseAndExpression() (PermissionQuery, error) {
	left, err := p.parsePrimary()
	if err != nil {
		return PermissionQuery{}, err
	}

	// Handle AND operators (higher precedence than OR)
	for p.currentToken.typ == and {
		p.readToken() // consume AND
		right, err := p.parsePrimary()
		if err != nil {
			return PermissionQuery{}, err
		}
		left = And(left, right)
	}

	return left, nil
}

// parsePrimary parses primary expressions, which are the building blocks of the grammar.
//
// Primary expressions can be:
//   - Permission identifiers (e.g., "api.key1.read_key")
//   - Parenthesized expressions (e.g., "(perm1 OR perm2)")
//
// For permission identifiers, the method creates a simple [PermissionQuery] with
// the permission string as the value.
//
// For parenthesized expressions, the method recursively parses the inner expression
// and ensures proper parenthesis matching. Empty parentheses are rejected as invalid.
//
// Error handling:
//   - Empty parentheses produce a specific error message
//   - Unmatched parentheses are detected and reported with position information
//   - Unexpected tokens (including EOF) generate descriptive error messages
//   - Lexer errors (invalid characters) are propagated up
//
// Returns the parsed [PermissionQuery] or an error with position information.
func (p *parser) parsePrimary() (PermissionQuery, error) {
	switch p.currentToken.typ {
	case permission:
		// Simple permission
		perm := p.currentToken.value
		p.readToken() // consume permission
		return S(perm), nil

	case lparen:
		// Parenthesized expression
		p.readToken() // consume (

		if p.currentToken.typ == rparen {
			return PermissionQuery{}, fault.New(
				fmt.Sprintf("empty parentheses at position %d in query", p.currentToken.pos),
				fault.Code(codes.User.BadRequest.PermissionsQuerySyntaxError.URN()),
				fault.Public(fmt.Sprintf("Empty parentheses found at position %d. Parentheses must contain at least one permission or expression.", p.currentToken.pos)),
			)
		}

		expr, err := p.parseExpression()
		if err != nil {
			return PermissionQuery{}, err
		}

		if p.currentToken.typ != rparen {
			return PermissionQuery{}, fault.New(
				fmt.Sprintf("missing closing parenthesis at position %d, found '%s'", p.currentToken.pos, p.currentToken.value),
				fault.Code(codes.User.BadRequest.PermissionsQuerySyntaxError.URN()),
				fault.Public(fmt.Sprintf("Missing closing parenthesis ')' at position %d. Every opening parenthesis must have a matching closing parenthesis.", p.currentToken.pos)),
			)
		}
		p.readToken() // consume )

		return expr, nil

	case eof:
		return PermissionQuery{}, fault.New(
			"reached end of input while expecting a permission or opening parenthesis",
			fault.Code(codes.User.BadRequest.PermissionsQuerySyntaxError.URN()),
			fault.Public("Unexpected end of query. Expected a permission identifier or opening parenthesis."),
		)

	case errorToken:
		// The lexer has already created a structured error, just return it
		return PermissionQuery{}, fmt.Errorf("%s", p.currentToken.value)

	default:
		return PermissionQuery{}, fault.New(
			fmt.Sprintf("unexpected token '%s' at position %d, expected permission or opening parenthesis", p.currentToken.value, p.currentToken.pos),
			fault.Code(codes.User.BadRequest.PermissionsQuerySyntaxError.URN()),
			fault.Public(fmt.Sprintf("Unexpected token '%s' at position %d. Expected a permission identifier or opening parenthesis.", p.currentToken.value, p.currentToken.pos)),
		)
	}
}

// parse performs the complete parsing of a permission query string.
//
// This is the main entry point for parsing. It orchestrates the complete
// parsing process by:
//  1. Checking for empty input
//  2. Parsing the complete expression
//  3. Verifying that all tokens were consumed
//
// The method ensures that the entire input is consumed during parsing.
// Any remaining tokens after parsing completes indicate a syntax error.
//
// Error conditions:
//   - Empty query (no tokens except EOF)
//   - Syntax errors during expression parsing
//   - Unconsumed tokens after parsing completes
//
// Returns the complete [PermissionQuery] AST representing the parsed expression,
// or an error with position information if parsing fails.
func (p *parser) parse() (PermissionQuery, error) {
	if p.currentToken.typ == eof {
		return PermissionQuery{}, fault.New(
			"query contains no tokens besides EOF",
			fault.Code(codes.User.BadRequest.PermissionsQuerySyntaxError.URN()),
			fault.Public("Unexpected end of input. Please provide a valid permission query."),
		)
	}

	expr, err := p.parseExpression()
	if err != nil {
		return PermissionQuery{}, err
	}

	// Check that we've consumed all tokens
	if p.currentToken.typ != eof {
		return PermissionQuery{}, fault.New(
			fmt.Sprintf("unexpected token '%s' at position %d after parsing completed", p.currentToken.value, p.currentToken.pos),
			fault.Code(codes.User.BadRequest.PermissionsQuerySyntaxError.URN()),
			fault.Public(fmt.Sprintf("Unexpected token '%s' at position %d. The query appears to have extra content after a valid expression.", p.currentToken.value, p.currentToken.pos)),
		)
	}

	return expr, nil
}

// parseQuery parses a permission query string with validation and complexity limits.
//
// This function provides the complete parsing pipeline including input validation,
// lexical analysis, syntactic parsing, and complexity checking. It enforces
// practical limits to prevent resource exhaustion and ensure reasonable performance.
//
// Complexity limits enforced:
//   - Maximum query length: 1000 characters (prevents excessive memory usage)
//
// The query string is automatically trimmed of leading and trailing whitespace.
// Empty or whitespace-only queries are rejected.
//
// Processing pipeline:
//  1. Input validation (length, emptiness)
//  2. Lexical analysis (tokenization)
//  3. Syntactic parsing (AST generation)
//
// Error conditions:
//   - Query exceeds 1000 characters
//   - Query is empty or whitespace-only
//   - Syntax errors in the query
//
// Returns the parsed [PermissionQuery] ready for use with [RBAC.EvaluatePermissions],
// or an error describing the validation or parsing failure.
func parseQuery(query string) (PermissionQuery, error) {
	// Parse the query
	p := newParser(query)
	result, err := p.parse()
	if err != nil {
		return PermissionQuery{}, err
	}

	return result, nil
}
