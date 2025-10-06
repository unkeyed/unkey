package resulttransformer

import (
	"context"
	"fmt"
	"strings"
)

// extractStringValue extracts a string from ch.Dynamic values returned by ClickHouse.
// ch.Dynamic wraps values in braces like "{id_123 }" so we strip them.
func extractStringValue(value any) string {
	// Convert to string - ClickHouse always returns ch.Dynamic
	str := fmt.Sprint(value)

	// ch.Dynamic wraps values in braces like "{id_123 }"
	// Strip the braces and whitespace
	str = strings.TrimSpace(str)
	if strings.HasPrefix(str, "{") && strings.HasSuffix(str, "}") {
		str = strings.TrimSpace(str[1 : len(str)-1])
	}

	return str
}

// ReverseResolver resolves actual ClickHouse IDs back to user-facing IDs
// Input: actual IDs from ClickHouse (e.g., ["ks_abc", "ks_def"])
// Output: map from actual ID to user ID (e.g., {"ks_abc": "api_123", "ks_def": "api_456"})
type ReverseResolver func(ctx context.Context, actualIDs []string) (map[string]string, error)

// ColumnConfig defines how to transform a column
type ColumnConfig struct {
	ActualColumn  string          // e.g., "key_space_id"
	VirtualColumn string          // e.g., "apiId"
	Resolver      ReverseResolver // Resolves actual IDs to user IDs
}

// ColumnMapping defines which result columns map to which actual columns
type ColumnMapping struct {
	ResultColumn string // The column name in results (e.g., "a" from "SELECT key_space_id AS a")
	ActualColumn string // The actual ClickHouse column (e.g., "key_space_id")
}

// Transformer transforms ClickHouse query results back to user-facing format
type Transformer struct {
	columns []ColumnConfig
}

// New creates a new result transformer
func New(columns []ColumnConfig) *Transformer {
	return &Transformer{
		columns: columns,
	}
}

// TransformWithMappings transforms query results using column mappings from the parser
func (t *Transformer) TransformWithMappings(ctx context.Context, results []map[string]any, mappings []ColumnMapping) ([]map[string]any, error) {
	if len(results) == 0 {
		return results, nil
	}

	// Build maps for O(1) lookups
	// Map: actual column -> result columns
	actualToResult := make(map[string][]string)
	for _, m := range mappings {
		actualToResult[m.ActualColumn] = append(actualToResult[m.ActualColumn], m.ResultColumn)
	}

	// Map: result column -> actual column
	resultToActual := make(map[string]string)
	for _, m := range mappings {
		resultToActual[m.ResultColumn] = m.ActualColumn
	}

	// Map: actual column -> config
	actualToConfig := make(map[string]*ColumnConfig)
	for i := range t.columns {
		actualToConfig[t.columns[i].ActualColumn] = &t.columns[i]
	}

	// Collect all IDs that need resolution per actual column
	idsToResolve := make(map[string]map[string]bool) // actual column -> set of IDs
	for _, col := range t.columns {
		idsToResolve[col.ActualColumn] = make(map[string]bool)
	}

	for _, row := range results {
		for _, col := range t.columns {
			// Check both the actual column name and any result columns that map to it
			resultColumns := actualToResult[col.ActualColumn]
			if len(resultColumns) == 0 {
				resultColumns = []string{col.ActualColumn}
			}

			for _, resultCol := range resultColumns {
				if value, exists := row[resultCol]; exists {
					strValue := extractStringValue(value)
					if strValue != "" {
						idsToResolve[col.ActualColumn][strValue] = true
					}
				}
			}
		}
	}

	// Resolve all IDs
	resolvedMappings := make(map[string]map[string]string) // actual column -> actual ID -> user ID
	for _, col := range t.columns {
		ids := idsToResolve[col.ActualColumn]
		if len(ids) == 0 {
			continue
		}

		// Convert set to slice
		idSlice := make([]string, 0, len(ids))
		for id := range ids {
			idSlice = append(idSlice, id)
		}

		// Resolve
		mapping, err := col.Resolver(ctx, idSlice)
		if err != nil {
			return nil, err
		}

		resolvedMappings[col.ActualColumn] = mapping
	}

	// Transform results
	transformed := make([]map[string]any, len(results))
	for i, row := range results {
		transformedRow := make(map[string]any)

		for resultColumn, value := range row {
			transformedValue := value

			// Find the actual column and config using maps (O(1) lookup)
			actualColumn, hasMappingEntry := resultToActual[resultColumn]
			if !hasMappingEntry {
				// No mapping found, check if it's directly an actual column
				actualColumn = resultColumn
			}

			// Check if we have a config for this actual column
			if _, hasConfig := actualToConfig[actualColumn]; hasConfig {
				strValue := extractStringValue(value)
				if mapping, exists := resolvedMappings[actualColumn]; exists {
					if userValue, found := mapping[strValue]; found {
						transformedValue = userValue
					}
				}
			}

			transformedRow[resultColumn] = transformedValue
		}

		transformed[i] = transformedRow
	}

	return transformed, nil
}
