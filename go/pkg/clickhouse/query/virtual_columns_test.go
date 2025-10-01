package query

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

// Mock resolver for tests that don't need actual resolution
func mockResolver(ctx context.Context, ids []string) (map[string]string, error) {
	result := make(map[string]string)
	for _, id := range ids {
		result[id] = "resolved_" + id
	}
	return result, nil
}

func TestExtractVirtualColumns(t *testing.T) {
	r := New(Config{
		WorkspaceID: "ws_123",
		VirtualColumns: map[string]VirtualColumn{
			"apiId":      {ActualColumn: "key_space_id"},
			"externalId": {ActualColumn: "identity_id"},
		},
	})

	tests := []struct {
		name     string
		query    string
		expected []VirtualColumnValue
	}{
		{
			name:  "single virtual column",
			query: "SELECT * FROM key_verifications WHERE apiId = 'api_123'",
			expected: []VirtualColumnValue{
				{
					VirtualColumn: "apiId",
					Value:         "api_123",
					ActualColumn:  "key_space_id",
				},
			},
		},
		{
			name:  "two virtual columns",
			query: "SELECT * FROM key_verifications WHERE apiId = 'api_123' AND externalId = 'user_456'",
			expected: []VirtualColumnValue{
				{
					VirtualColumn: "apiId",
					Value:         "api_123",
					ActualColumn:  "key_space_id",
				},
				{
					VirtualColumn: "externalId",
					Value:         "user_456",
					ActualColumn:  "identity_id",
				},
			},
		},
		{
			name:  "virtual column with other conditions",
			query: "SELECT * FROM key_verifications WHERE apiId = 'api_123' AND valid = true",
			expected: []VirtualColumnValue{
				{
					VirtualColumn: "apiId",
					Value:         "api_123",
					ActualColumn:  "key_space_id",
				},
			},
		},
		{
			name:     "no virtual columns",
			query:    "SELECT * FROM key_verifications WHERE valid = true",
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			virtualCols, err := r.ExtractVirtualColumns(tt.query)
			require.NoError(t, err)

			require.Equal(t, len(tt.expected), len(virtualCols))
			for i, expected := range tt.expected {
				require.Equal(t, expected.VirtualColumn, virtualCols[i].VirtualColumn)
				require.Equal(t, expected.Value, virtualCols[i].Value)
				require.Equal(t, expected.ActualColumn, virtualCols[i].ActualColumn)
			}
		})
	}
}

func TestRewriteWithVirtualColumns(t *testing.T) {
	r := New(Config{
		WorkspaceID: "ws_123",
		TableAliases: map[string]string{
			"key_verifications": "default.key_verifications_v1",
		},
		AllowedTables: []string{
			"default.key_verifications_v1",
		},
		VirtualColumns: map[string]VirtualColumn{
			"apiId":      {ActualColumn: "key_space_id"},
			"externalId": {ActualColumn: "identity_id"},
		},
	})

	tests := []struct {
		name        string
		query       string
		virtualCols []VirtualColumnValue
		checkOutput func(t *testing.T, output string)
	}{
		{
			name:  "replace apiId with key_space_id",
			query: "SELECT * FROM key_verifications WHERE apiId = 'api_123'",
			virtualCols: []VirtualColumnValue{
				{
					VirtualColumn: "apiId",
					Value:         "api_123",
					ActualColumn:  "key_space_id",
					ActualValue:   "keyauth_xyz", // This would come from database lookup
				},
			},
			checkOutput: func(t *testing.T, output string) {
				require.Contains(t, output, "key_space_id = 'keyauth_xyz'")
				require.NotContains(t, output, "apiId")
				require.NotContains(t, output, "api_123")
				require.Contains(t, output, "workspace_id = 'ws_123'")
			},
		},
		{
			name:  "replace externalId with identity_id",
			query: "SELECT * FROM key_verifications WHERE externalId = 'user_456'",
			virtualCols: []VirtualColumnValue{
				{
					VirtualColumn: "externalId",
					Value:         "user_456",
					ActualColumn:  "identity_id",
					ActualValue:   "identity_internal_789",
				},
			},
			checkOutput: func(t *testing.T, output string) {
				require.Contains(t, output, "identity_id = 'identity_internal_789'")
				require.NotContains(t, output, "externalId")
				require.NotContains(t, output, "user_456")
				require.Contains(t, output, "workspace_id = 'ws_123'")
			},
		},
		{
			name:  "replace both apiId and externalId",
			query: "SELECT * FROM key_verifications WHERE apiId = 'api_123' AND externalId = 'user_456'",
			virtualCols: []VirtualColumnValue{
				{
					VirtualColumn: "apiId",
					Value:         "api_123",
					ActualColumn:  "key_space_id",
					ActualValue:   "keyauth_xyz",
				},
				{
					VirtualColumn: "externalId",
					Value:         "user_456",
					ActualColumn:  "identity_id",
					ActualValue:   "identity_internal_789",
				},
			},
			checkOutput: func(t *testing.T, output string) {
				require.Contains(t, output, "key_space_id = 'keyauth_xyz'")
				require.Contains(t, output, "identity_id = 'identity_internal_789'")
				require.Contains(t, output, "workspace_id = 'ws_123'")
			},
		},
		{
			name:  "virtual column with other normal conditions",
			query: "SELECT * FROM key_verifications WHERE apiId = 'api_123' AND valid = true",
			virtualCols: []VirtualColumnValue{
				{
					VirtualColumn: "apiId",
					Value:         "api_123",
					ActualColumn:  "key_space_id",
					ActualValue:   "keyauth_xyz",
				},
			},
			checkOutput: func(t *testing.T, output string) {
				require.Contains(t, output, "key_space_id = 'keyauth_xyz'")
				require.Contains(t, output, "valid = true")
				require.Contains(t, output, "workspace_id = 'ws_123'")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := r.RewriteWithVirtualColumns(tt.query, tt.virtualCols)
			require.NoError(t, err)
			tt.checkOutput(t, output)
		})
	}
}

func TestVirtualColumnsIntegration(t *testing.T) {
	// This test shows the full workflow of using virtual columns

	r := New(Config{
		WorkspaceID: "ws_victim",
		TableAliases: map[string]string{
			"key_verifications": "default.key_verifications_v1",
		},
		AllowedTables: []string{
			"default.key_verifications_v1",
		},
		VirtualColumns: map[string]VirtualColumn{
			"apiId":      {ActualColumn: "key_space_id"},
			"externalId": {ActualColumn: "identity_id"},
		},
	})

	userQuery := "SELECT COUNT(*) as total FROM key_verifications WHERE apiId = 'api_123' AND externalId = 'user_external'"

	// Step 1: Extract virtual columns
	virtualCols, err := r.ExtractVirtualColumns(userQuery)
	require.NoError(t, err)
	require.Len(t, virtualCols, 2)

	// Step 2: Simulate database lookups (in real code, this would query the database)
	for i := range virtualCols {
		if virtualCols[i].VirtualColumn == "apiId" {
			// Simulate lookup: apiId 'api_123' → keyAuthId 'keyauth_real_id'
			virtualCols[i].ActualValue = "keyauth_real_id"
		}
		if virtualCols[i].VirtualColumn == "externalId" {
			// Simulate lookup: externalId 'user_external' → identity.id 'identity_real_id'
			virtualCols[i].ActualValue = "identity_real_id"
		}
	}

	// Step 3: Rewrite query with resolved values
	rewritten, err := r.RewriteWithVirtualColumns(userQuery, virtualCols)
	require.NoError(t, err)

	// Step 4: Verify output
	require.Contains(t, rewritten, "workspace_id = 'ws_victim'")
	require.Contains(t, rewritten, "key_space_id = 'keyauth_real_id'")
	require.Contains(t, rewritten, "identity_id = 'identity_real_id'")
	require.NotContains(t, rewritten, "apiId")
	require.NotContains(t, rewritten, "externalId")

	t.Logf("Original query: %s", userQuery)
	t.Logf("Rewritten query: %s", rewritten)
}

func TestVirtualColumnsWithInjectionAttempt(t *testing.T) {
	// Ensure virtual column rewriting doesn't create security issues

	r := New(Config{
		WorkspaceID: "ws_victim",
		TableAliases: map[string]string{
			"key_verifications": "default.key_verifications_v1",
		},
		AllowedTables: []string{
			"default.key_verifications_v1",
		},
		VirtualColumns: map[string]VirtualColumn{
			"apiId": {ActualColumn: "key_space_id"},
		},
	})

	// User tries to bypass workspace isolation using virtual column
	userQuery := "SELECT * FROM key_verifications WHERE apiId = 'api_attacker'"

	virtualCols, err := r.ExtractVirtualColumns(userQuery)
	require.NoError(t, err)
	require.Len(t, virtualCols, 1)

	// Even if they provide an attacker's API, we still inject workspace filter
	virtualCols[0].ActualValue = "keyauth_attacker"

	rewritten, err := r.RewriteWithVirtualColumns(userQuery, virtualCols)
	require.NoError(t, err)

	// Workspace filter should be present AND at the beginning
	require.Contains(t, rewritten, "workspace_id = 'ws_victim'")
	require.Contains(t, rewritten, "key_space_id = 'keyauth_attacker'")

	// The query will only return data for ws_victim that also matches keyauth_attacker
	// If keyauth_attacker doesn't belong to ws_victim, returns empty (safe!)
}

func TestExtractVirtualColumnsWithIN(t *testing.T) {
	r := New(Config{
		WorkspaceID: "ws_123",
		VirtualColumns: map[string]VirtualColumn{
			"apiId":      {ActualColumn: "key_space_id"},
			"externalId": {ActualColumn: "identity_id"},
		},
	})

	tests := []struct {
		name     string
		query    string
		expected []VirtualColumnValue
	}{
		{
			name:  "single virtual column with IN clause",
			query: "SELECT * FROM key_verifications WHERE apiId IN ('api_123', 'api_456')",
			expected: []VirtualColumnValue{
				{
					VirtualColumn: "apiId",
					Values:        []string{"api_123", "api_456"},
					ActualColumn:  "key_space_id",
				},
			},
		},
		{
			name:  "multiple virtual columns with IN clauses",
			query: "SELECT * FROM key_verifications WHERE apiId IN ('api_123', 'api_456') AND externalId IN ('user_1', 'user_2', 'user_3')",
			expected: []VirtualColumnValue{
				{
					VirtualColumn: "apiId",
					Values:        []string{"api_123", "api_456"},
					ActualColumn:  "key_space_id",
				},
				{
					VirtualColumn: "externalId",
					Values:        []string{"user_1", "user_2", "user_3"},
					ActualColumn:  "identity_id",
				},
			},
		},
		{
			name:  "mix of = and IN clauses",
			query: "SELECT * FROM key_verifications WHERE apiId IN ('api_123', 'api_456') AND valid = true",
			expected: []VirtualColumnValue{
				{
					VirtualColumn: "apiId",
					Values:        []string{"api_123", "api_456"},
					ActualColumn:  "key_space_id",
				},
			},
		},
		{
			name:  "IN clause with single value",
			query: "SELECT * FROM key_verifications WHERE apiId IN ('api_123')",
			expected: []VirtualColumnValue{
				{
					VirtualColumn: "apiId",
					Values:        []string{"api_123"},
					ActualColumn:  "key_space_id",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			virtualCols, err := r.ExtractVirtualColumns(tt.query)
			require.NoError(t, err)

			require.Equal(t, len(tt.expected), len(virtualCols))
			for i, expected := range tt.expected {
				require.Equal(t, expected.VirtualColumn, virtualCols[i].VirtualColumn)
				require.Equal(t, expected.Values, virtualCols[i].Values)
				require.Equal(t, expected.ActualColumn, virtualCols[i].ActualColumn)
			}
		})
	}
}

func TestRewriteWithVirtualColumnsIN(t *testing.T) {
	r := New(Config{
		WorkspaceID: "ws_123",
		TableAliases: map[string]string{
			"key_verifications": "default.key_verifications_v1",
		},
		AllowedTables: []string{
			"default.key_verifications_v1",
		},
		VirtualColumns: map[string]VirtualColumn{
			"apiId":      {ActualColumn: "key_space_id"},
			"externalId": {ActualColumn: "identity_id"},
		},
	})

	tests := []struct {
		name        string
		query       string
		virtualCols []VirtualColumnValue
		checkOutput func(t *testing.T, output string)
	}{
		{
			name:  "replace apiId IN clause with key_space_id values",
			query: "SELECT * FROM key_verifications WHERE apiId IN ('api_123', 'api_456')",
			virtualCols: []VirtualColumnValue{
				{
					VirtualColumn: "apiId",
					Values:        []string{"api_123", "api_456"},
					ActualColumn:  "key_space_id",
					ActualValues:  []string{"keyauth_xyz", "keyauth_abc"},
				},
			},
			checkOutput: func(t *testing.T, output string) {
				require.Contains(t, output, "key_space_id in ('keyauth_xyz', 'keyauth_abc')")
				require.NotContains(t, output, "apiId")
				require.NotContains(t, output, "api_123")
				require.NotContains(t, output, "api_456")
				require.Contains(t, output, "workspace_id = 'ws_123'")
			},
		},
		{
			name:  "replace multiple IN clauses",
			query: "SELECT * FROM key_verifications WHERE apiId IN ('api_123', 'api_456') AND externalId IN ('user_1', 'user_2')",
			virtualCols: []VirtualColumnValue{
				{
					VirtualColumn: "apiId",
					Values:        []string{"api_123", "api_456"},
					ActualColumn:  "key_space_id",
					ActualValues:  []string{"keyauth_xyz", "keyauth_abc"},
				},
				{
					VirtualColumn: "externalId",
					Values:        []string{"user_1", "user_2"},
					ActualColumn:  "identity_id",
					ActualValues:  []string{"identity_internal_1", "identity_internal_2"},
				},
			},
			checkOutput: func(t *testing.T, output string) {
				require.Contains(t, output, "key_space_id in ('keyauth_xyz', 'keyauth_abc')")
				require.Contains(t, output, "identity_id in ('identity_internal_1', 'identity_internal_2')")
				require.Contains(t, output, "workspace_id = 'ws_123'")
			},
		},
		{
			name:  "IN clause with single value",
			query: "SELECT * FROM key_verifications WHERE apiId IN ('api_123')",
			virtualCols: []VirtualColumnValue{
				{
					VirtualColumn: "apiId",
					Values:        []string{"api_123"},
					ActualColumn:  "key_space_id",
					ActualValues:  []string{"keyauth_xyz"},
				},
			},
			checkOutput: func(t *testing.T, output string) {
				require.Contains(t, output, "key_space_id in ('keyauth_xyz')")
				require.Contains(t, output, "workspace_id = 'ws_123'")
			},
		},
		{
			name:  "mix of IN and = comparisons",
			query: "SELECT * FROM key_verifications WHERE apiId IN ('api_123', 'api_456') AND externalId = 'user_1'",
			virtualCols: []VirtualColumnValue{
				{
					VirtualColumn: "apiId",
					Values:        []string{"api_123", "api_456"},
					ActualColumn:  "key_space_id",
					ActualValues:  []string{"keyauth_xyz", "keyauth_abc"},
				},
				{
					VirtualColumn: "externalId",
					Value:         "user_1",
					ActualColumn:  "identity_id",
					ActualValue:   "identity_internal_1",
				},
			},
			checkOutput: func(t *testing.T, output string) {
				require.Contains(t, output, "key_space_id in ('keyauth_xyz', 'keyauth_abc')")
				require.Contains(t, output, "identity_id = 'identity_internal_1'")
				require.Contains(t, output, "workspace_id = 'ws_123'")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := r.RewriteWithVirtualColumns(tt.query, tt.virtualCols)
			require.NoError(t, err)
			tt.checkOutput(t, output)
		})
	}
}

func TestVirtualColumnsINIntegration(t *testing.T) {
	// This test shows the full workflow of using virtual columns with IN clauses

	r := New(Config{
		WorkspaceID: "ws_victim",
		TableAliases: map[string]string{
			"key_verifications": "default.key_verifications_v1",
		},
		AllowedTables: []string{
			"default.key_verifications_v1",
		},
		VirtualColumns: map[string]VirtualColumn{
			"apiId":      {ActualColumn: "key_space_id"},
			"externalId": {ActualColumn: "identity_id"},
		},
	})

	userQuery := "SELECT COUNT(*) as total FROM key_verifications WHERE apiId IN ('api_123', 'api_456') AND externalId IN ('user_ext_1', 'user_ext_2')"

	// Step 1: Extract virtual columns
	virtualCols, err := r.ExtractVirtualColumns(userQuery)
	require.NoError(t, err)
	require.Len(t, virtualCols, 2)

	// Step 2: Simulate database lookups (in real code, this would query the database)
	for i := range virtualCols {
		if virtualCols[i].VirtualColumn == "apiId" {
			// Simulate lookup: apiId 'api_123' → keyAuthId 'keyauth_real_id_1'
			//                  apiId 'api_456' → keyAuthId 'keyauth_real_id_2'
			virtualCols[i].ActualValues = []string{"keyauth_real_id_1", "keyauth_real_id_2"}
		}
		if virtualCols[i].VirtualColumn == "externalId" {
			// Simulate lookup: externalId 'user_ext_1' → identity.id 'identity_real_id_1'
			//                  externalId 'user_ext_2' → identity.id 'identity_real_id_2'
			virtualCols[i].ActualValues = []string{"identity_real_id_1", "identity_real_id_2"}
		}
	}

	// Step 3: Rewrite query with resolved values
	rewritten, err := r.RewriteWithVirtualColumns(userQuery, virtualCols)
	require.NoError(t, err)

	// Step 4: Verify output
	require.Contains(t, rewritten, "workspace_id = 'ws_victim'")
	require.Contains(t, rewritten, "key_space_id in ('keyauth_real_id_1', 'keyauth_real_id_2')")
	require.Contains(t, rewritten, "identity_id in ('identity_real_id_1', 'identity_real_id_2')")
	require.NotContains(t, rewritten, "apiId")
	require.NotContains(t, rewritten, "externalId")

	t.Logf("Original query: %s", userQuery)
	t.Logf("Rewritten query: %s", rewritten)
}
