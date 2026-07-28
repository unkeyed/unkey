package mysql

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

const caseSensitiveCollation = "COLLATE utf8mb4_0900_as_cs"

func TestSchemaCaseSensitiveVarchars(t *testing.T) {
	entries, err := os.ReadDir("schema")
	require.NoError(t, err)

	varcharCount := 0

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".sql" {
			continue
		}

		path := filepath.Join("schema", entry.Name())
		data, readErr := os.ReadFile(path)
		require.NoError(t, readErr)

		for lineNumber, line := range strings.Split(string(data), "\n") {
			if !strings.Contains(strings.ToLower(line), "varchar(") {
				continue
			}

			varcharCount++
			require.Equal(t, 1, strings.Count(line, caseSensitiveCollation), "%s:%d", path, lineNumber+1)
			require.Equal(t, 1, strings.Count(strings.ToUpper(line), " COLLATE "), "%s:%d", path, lineNumber+1)
		}
	}

	require.Positive(t, varcharCount, "schema must contain at least one VARCHAR column")
}
