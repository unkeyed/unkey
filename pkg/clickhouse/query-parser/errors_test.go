package queryparser

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
)

func TestParser_ErrorCodes(t *testing.T) {
	tests := []struct {
		name          string
		config        Config
		query         string
		expectedCode  codes.URN
		expectedError string
	}{
		{
			name: "invalid SQL syntax",
			config: Config{
				WorkspaceID: "ws_123",
				AllowedTables: []string{
					"default.keys_v2",
				},
			},
			query:         "SELECT * FROM @@@",
			expectedCode:  codes.User.BadRequest.InvalidAnalyticsQuery.URN(),
			expectedError: "Invalid SQL syntax",
		},
		{
			name: "invalid table",
			config: Config{
				WorkspaceID: "ws_123",
				AllowedTables: []string{
					"default.keys_v2",
				},
			},
			query:         "SELECT * FROM system.tables",
			expectedCode:  codes.User.BadRequest.InvalidAnalyticsTable.URN(),
			expectedError: "Access to table 'system.tables' is not allowed",
		},
		{
			name: "invalid function",
			config: Config{
				WorkspaceID: "ws_123",
				AllowedTables: []string{
					"default.keys_v2",
				},
			},
			query:         "SELECT file('test') FROM default.keys_v2",
			expectedCode:  codes.User.BadRequest.InvalidAnalyticsFunction.URN(),
			expectedError: "Function 'file' is not allowed",
		},
		{
			name: "query not supported",
			config: Config{
				WorkspaceID: "ws_123",
				AllowedTables: []string{
					"default.keys_v2",
				},
			},
			query:         "INSERT INTO default.keys_v2 VALUES (1)",
			expectedCode:  codes.User.BadRequest.InvalidAnalyticsQueryType.URN(),
			expectedError: "Only SELECT queries are allowed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewParser(tt.config)
			_, err := parser.Parse(context.Background(), tt.query)

			require.Error(t, err)

			// Check error code
			code, ok := fault.GetCode(err)
			require.True(t, ok, "Expected error to have a code")
			require.Equal(t, tt.expectedCode, code)

			// Check public message
			publicMsg := fault.UserFacingMessage(err)
			require.Contains(t, publicMsg, tt.expectedError)
		})
	}
}
