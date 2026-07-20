package handler

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	queryparser "github.com/unkeyed/unkey/pkg/clickhouse/query-parser"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
)

func TestVerificationParserConfigRejectsJoins(t *testing.T) {
	parser := queryparser.NewParser(verificationParserConfig())
	_, err := parser.Parse(context.Background(), "SELECT * FROM key_verifications_v1 a JOIN key_verifications_per_hour_v1 b ON a.key_id = b.key_id")
	require.Error(t, err)
	code, ok := fault.GetCode(err)
	require.True(t, ok)
	require.Equal(t, codes.User.BadRequest.InvalidAnalyticsQuery.URN(), code)
	require.Equal(t, "JOIN queries are not supported", fault.UserFacingMessage(err))
}

func TestVerificationParserConfigRequiresPublicAliases(t *testing.T) {
	parser := queryparser.NewParser(verificationParserConfig())

	for _, table := range verificationTables {
		publicAlias := table.alias
		t.Run(publicAlias, func(t *testing.T) {
			_, err := parser.Parse(context.Background(), "SELECT * FROM "+publicAlias)
			require.NoError(t, err)
		})
	}

	for name, query := range map[string]string{
		"direct physical source": "SELECT * FROM default.key_verifications_raw_v2",
		"physical source in CTE": "WITH hidden AS (SELECT * FROM default.key_verifications_raw_v2) SELECT * FROM hidden",
	} {
		t.Run(name, func(t *testing.T) {
			_, err := parser.Parse(context.Background(), query)
			require.Error(t, err)
			code, ok := fault.GetCode(err)
			require.True(t, ok)
			require.Equal(t, codes.User.BadRequest.InvalidAnalyticsTable.URN(), code)
		})
	}
}
