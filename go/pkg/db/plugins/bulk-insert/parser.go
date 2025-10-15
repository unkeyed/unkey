package main

import (
	"strings"

	"github.com/sqlc-dev/plugin-sdk-go/plugin"
)

// SQLParser handles parsing of SQL INSERT queries.
type SQLParser struct{}

// ParsedQuery represents a parsed INSERT query.
type ParsedQuery struct {
	InsertPart             string
	ValuesPart             string
	OnDuplicateKeyUpdate   string
	ValuesPlaceholderCount int
}

// NewSQLParser creates a new SQL parser.
func NewSQLParser() *SQLParser {
	return &SQLParser{}
}

// Parse parses an INSERT query and extracts its components.
func (p *SQLParser) Parse(query *plugin.Query) *ParsedQuery {
	// Get the actual SQL query text
	originalSQL := query.Text
	if originalSQL == "" {
		originalSQL = query.Cmd
	}
	insertPart, valuesPart := p.parseInsertQuery(originalSQL)
	onDuplicateKeyUpdate := p.extractOnDuplicateKeyUpdate(originalSQL)
	placeholderCount := p.countPlaceholders(valuesPart)
	return &ParsedQuery{
		InsertPart:             insertPart,
		ValuesPart:             valuesPart,
		OnDuplicateKeyUpdate:   onDuplicateKeyUpdate,
		ValuesPlaceholderCount: placeholderCount,
	}
}

// parseInsertQuery separates the INSERT part from the VALUES part.
func (p *SQLParser) parseInsertQuery(query string) (string, string) {
	// Remove SQL comments and normalize whitespace
	cleanQuery := p.cleanSQL(query)

	// Don't remove backticks - they're needed for reserved keywords like `limit`
	upperQuery := strings.ToUpper(cleanQuery)
	valuesIndex := strings.Index(upperQuery, " VALUES")

	if valuesIndex == -1 {
		return cleanQuery, "()"
	}

	insertPart := strings.TrimSpace(cleanQuery[:valuesIndex])
	valuesPart := p.extractValuesClause(cleanQuery, valuesIndex)

	return insertPart, valuesPart
}

// cleanSQL removes comments and normalizes whitespace.
func (p *SQLParser) cleanSQL(query string) string {
	lines := strings.Split(query, "\n")
	var cleanLines []string

	for _, line := range lines {
		// Remove line comments
		if idx := strings.Index(line, "--"); idx >= 0 {
			line = line[:idx]
		}
		line = strings.TrimSpace(line)
		if line != "" {
			cleanLines = append(cleanLines, line)
		}
	}

	return strings.Join(cleanLines, " ")
}

// extractValuesClause extracts the VALUES clause from the query.
func (p *SQLParser) extractValuesClause(cleanQuery string, valuesIndex int) string {
	// Find the values clause
	valuesStart := valuesIndex + 7 // len(" VALUES")

	// Skip whitespace
	for valuesStart < len(cleanQuery) && cleanQuery[valuesStart] == ' ' {
		valuesStart++
	}

	if valuesStart >= len(cleanQuery) {
		return "()"
	}

	// Find matching parentheses
	if cleanQuery[valuesStart] == '(' {
		parenCount := 1
		for j := valuesStart + 1; j < len(cleanQuery); j++ {
			if cleanQuery[j] == '(' {
				parenCount++
			} else if cleanQuery[j] == ')' {
				parenCount--
				if parenCount == 0 {
					return strings.TrimSpace(cleanQuery[valuesStart : j+1])
				}
			}
		}
	}

	return "()"
}

// extractOnDuplicateKeyUpdate extracts the ON DUPLICATE KEY UPDATE clause if present.
func (p *SQLParser) extractOnDuplicateKeyUpdate(originalSQL string) string {
	upperSQL := strings.ToUpper(originalSQL)
	if idx := strings.Index(upperSQL, "ON DUPLICATE KEY UPDATE"); idx != -1 {
		return strings.TrimSpace(originalSQL[idx:])
	}
	return ""
}

// countPlaceholders counts the number of ? placeholders in the values clause.
func (p *SQLParser) countPlaceholders(valuesPart string) int {
	count := 0
	for _, ch := range valuesPart {
		if ch == '?' {
			count++
		}
	}
	return count
}
