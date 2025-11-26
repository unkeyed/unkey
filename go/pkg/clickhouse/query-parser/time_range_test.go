package queryparser

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
)

func TestParser_ValidateTimeRange_WithRetention(t *testing.T) {
	tests := []struct {
		name          string
		query         string
		retentionDays int32
		shouldPass    bool
		errorContains string
	}{
		{
			name:          "simple query within retention (5 days with 7 day retention)",
			query:         "SELECT * FROM key_verifications_per_hour_v1 WHERE time >= now() - INTERVAL 5 DAY",
			retentionDays: 7,
			shouldPass:    true,
		},
		{
			name:          "simple query exceeds retention (90 days with 7 day retention)",
			query:         "SELECT * FROM key_verifications_per_hour_v1 WHERE time >= now() - INTERVAL 90 DAY",
			retentionDays: 7,
			shouldPass:    false,
			errorContains: "retention period of 7 days",
		},
		{
			name:          "nested toStartOfHour within retention",
			query:         "SELECT * FROM key_verifications_per_hour_v1 WHERE time >= toStartOfHour(now() - INTERVAL 5 DAY)",
			retentionDays: 7,
			shouldPass:    true,
		},
		{
			name:          "nested toStartOfHour exceeds retention",
			query:         "SELECT * FROM key_verifications_per_hour_v1 WHERE time >= toStartOfHour(now() - INTERVAL 90 DAY)",
			retentionDays: 7,
			shouldPass:    false,
			errorContains: "retention period of 7 days",
		},
		{
			name:          "nested toStartOfDay within retention",
			query:         "SELECT * FROM key_verifications_per_hour_v1 WHERE time >= toStartOfDay(now() - INTERVAL 6 DAY)",
			retentionDays: 7,
			shouldPass:    true,
		},
		{
			name:          "nested toStartOfDay exceeds retention",
			query:         "SELECT * FROM key_verifications_per_hour_v1 WHERE time >= toStartOfDay(now() - INTERVAL 30 DAY)",
			retentionDays: 7,
			shouldPass:    false,
			errorContains: "retention period of 7 days",
		},
		{
			name:          "date_trunc with nested interval within retention",
			query:         "SELECT * FROM key_verifications_per_hour_v1 WHERE time >= date_trunc('hour', now() - INTERVAL 5 DAY)",
			retentionDays: 7,
			shouldPass:    true,
		},
		{
			name:          "date_trunc with nested interval exceeds retention",
			query:         "SELECT * FROM key_verifications_per_hour_v1 WHERE time >= date_trunc('hour', now() - INTERVAL 90 DAY)",
			retentionDays: 7,
			shouldPass:    false,
			errorContains: "retention period of 7 days",
		},
		{
			name:          "toDate within retention",
			query:         "SELECT * FROM key_verifications_per_hour_v1 WHERE time >= toDate(now() - INTERVAL 3 DAY)",
			retentionDays: 7,
			shouldPass:    true,
		},
		{
			name:          "toDateTime exceeds retention",
			query:         "SELECT * FROM key_verifications_per_hour_v1 WHERE time >= toDateTime(now() - INTERVAL 60 DAY)",
			retentionDays: 7,
			shouldPass:    false,
			errorContains: "retention period of 7 days",
		},
		{
			name:          "now64 within retention",
			query:         "SELECT * FROM key_verifications_per_hour_v1 WHERE time >= now64() - INTERVAL 5 DAY",
			retentionDays: 7,
			shouldPass:    true,
		},
		{
			name:          "today() within retention",
			query:         "SELECT * FROM key_verifications_per_hour_v1 WHERE time >= today() - INTERVAL 5 DAY",
			retentionDays: 7,
			shouldPass:    true,
		},
		{
			name:          "complex WHERE with AND - both conditions within retention",
			query:         "SELECT * FROM key_verifications_per_hour_v1 WHERE time >= now() - INTERVAL 5 DAY AND time <= now()",
			retentionDays: 7,
			shouldPass:    true,
		},
		{
			name:          "complex WHERE with AND - one condition exceeds retention",
			query:         "SELECT * FROM key_verifications_per_hour_v1 WHERE time >= now() - INTERVAL 90 DAY AND time <= now()",
			retentionDays: 7,
			shouldPass:    false,
			errorContains: "retention period of 7 days",
		},
		{
			name:          "query with no time filter (should auto-inject)",
			query:         "SELECT * FROM key_verifications_per_hour_v1 WHERE key_space_id = 'ks_123'",
			retentionDays: 7,
			shouldPass:    true,
		},
		{
			name:          "query with no time filter and no retention limit",
			query:         "SELECT * FROM key_verifications_per_hour_v1 WHERE key_space_id = 'ks_123'",
			retentionDays: 0, // No retention limit
			shouldPass:    true,
		},
		{
			name:          "fromUnixTimestamp64Milli with old timestamp",
			query:         "SELECT * FROM key_verifications_per_hour_v1 WHERE time >= fromUnixTimestamp64Milli(1609459200000)",
			retentionDays: 7,
			shouldPass:    false,
			errorContains: "retention period of 7 days",
		},
		{
			name:          "toStartOfMinute within retention",
			query:         "SELECT * FROM key_verifications_per_hour_v1 WHERE time >= toStartOfMinute(now() - INTERVAL 3 DAY)",
			retentionDays: 7,
			shouldPass:    true,
		},
		{
			name:          "toStartOfWeek within retention",
			query:         "SELECT * FROM key_verifications_per_hour_v1 WHERE time >= toStartOfWeek(now() - INTERVAL 6 DAY)",
			retentionDays: 7,
			shouldPass:    true,
		},
		{
			name:          "toStartOfMonth exceeds retention",
			query:         "SELECT * FROM key_verifications_per_hour_v1 WHERE time >= toStartOfMonth(now() - INTERVAL 60 DAY)",
			retentionDays: 7,
			shouldPass:    false,
			errorContains: "retention period of 7 days",
		},
		{
			name:          "toStartOfQuarter exceeds retention",
			query:         "SELECT * FROM key_verifications_per_hour_v1 WHERE time >= toStartOfQuarter(now() - INTERVAL 120 DAY)",
			retentionDays: 7,
			shouldPass:    false,
			errorContains: "retention period of 7 days",
		},
		{
			name:          "toStartOfYear exceeds retention",
			query:         "SELECT * FROM key_verifications_per_hour_v1 WHERE time >= toStartOfYear(now() - INTERVAL 365 DAY)",
			retentionDays: 7,
			shouldPass:    false,
			errorContains: "retention period of 7 days",
		},
		{
			name:          "BETWEEN within retention",
			query:         "SELECT * FROM key_verifications_per_hour_v1 WHERE time BETWEEN now() - INTERVAL 5 DAY AND now()",
			retentionDays: 7,
			shouldPass:    true,
		},
		{
			name:          "BETWEEN exceeds retention",
			query:         "SELECT * FROM key_verifications_per_hour_v1 WHERE time BETWEEN now() - INTERVAL 90 DAY AND now()",
			retentionDays: 7,
			shouldPass:    false,
			errorContains: "retention period of 7 days",
		},
		{
			name:          "BETWEEN at exact retention limit",
			query:         "SELECT * FROM key_verifications_per_hour_v1 WHERE time BETWEEN now() - INTERVAL 7 DAY AND now()",
			retentionDays: 7,
			shouldPass:    true,
		},
		{
			name:          "BETWEEN with nested functions within retention",
			query:         "SELECT * FROM key_verifications_per_hour_v1 WHERE time BETWEEN toStartOfDay(now() - INTERVAL 3 DAY) AND now()",
			retentionDays: 7,
			shouldPass:    true,
		},
		{
			name:          "BETWEEN with nested functions exceeds retention",
			query:         "SELECT * FROM key_verifications_per_hour_v1 WHERE time BETWEEN toStartOfDay(now() - INTERVAL 60 DAY) AND now()",
			retentionDays: 7,
			shouldPass:    false,
			errorContains: "retention period of 7 days",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewParser(Config{
				WorkspaceID: "test_ws",
				TableAliases: map[string]string{
					"key_verifications_per_hour_v1": "default.key_verifications_per_hour_v2",
				},
				AllowedTables: []string{
					"default.key_verifications_per_hour_v2",
				},
				MaxQueryRangeDays: tt.retentionDays,
				Logger:            logging.NewNoop(),
			})

			_, err := parser.Parse(context.Background(), tt.query)

			if tt.shouldPass {
				require.NoError(t, err, "expected query to pass validation")
			} else {
				require.Error(t, err, "expected query to fail validation")
				if tt.errorContains != "" {
					require.Contains(t, err.Error(), tt.errorContains)
				}
			}
		})
	}
}

func TestParser_ValidateTimeRange_NoRetentionLimit(t *testing.T) {
	parser := NewParser(Config{
		WorkspaceID: "test_ws",
		TableAliases: map[string]string{
			"key_verifications_per_hour_v1": "default.key_verifications_per_hour_v2",
		},
		AllowedTables: []string{
			"default.key_verifications_per_hour_v2",
		},
		MaxQueryRangeDays: 0, // No limit
		Logger:            logging.NewNoop(),
	})

	tests := []struct {
		name  string
		query string
	}{
		{
			name:  "any time range allowed when retention is 0",
			query: "SELECT * FROM key_verifications_per_hour_v1 WHERE time >= now() - INTERVAL 365 DAY",
		},
		{
			name:  "very old data allowed",
			query: "SELECT * FROM key_verifications_per_hour_v1 WHERE time >= now() - INTERVAL 1000 DAY",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parser.Parse(context.Background(), tt.query)
			require.NoError(t, err, "queries should pass when retention limit is 0")
		})
	}
}

func TestParser_ValidateTimeRange_DifferentOperators(t *testing.T) {
	tests := []struct {
		name          string
		query         string
		shouldPass    bool
		errorContains string
	}{
		{
			name:       "time >= (greater than or equal)",
			query:      "SELECT * FROM key_verifications_per_hour_v1 WHERE time >= now() - INTERVAL 90 DAY",
			shouldPass: false,
		},
		{
			name:       "time > (greater than)",
			query:      "SELECT * FROM key_verifications_per_hour_v1 WHERE time > now() - INTERVAL 90 DAY",
			shouldPass: false,
		},
		{
			name:       "time <= (less than or equal) - should pass, querying recent data",
			query:      "SELECT * FROM key_verifications_per_hour_v1 WHERE time <= now()",
			shouldPass: true,
		},
		{
			name:       "time < (less than) - should pass, querying recent data",
			query:      "SELECT * FROM key_verifications_per_hour_v1 WHERE time < now() + INTERVAL 1 DAY",
			shouldPass: true,
		},
		{
			name:       "time = (equals) with old timestamp",
			query:      "SELECT * FROM key_verifications_per_hour_v1 WHERE time = now() - INTERVAL 90 DAY",
			shouldPass: false,
		},
	}

	parser := NewParser(Config{
		WorkspaceID: "test_ws",
		TableAliases: map[string]string{
			"key_verifications_per_hour_v1": "default.key_verifications_per_hour_v2",
		},
		AllowedTables: []string{
			"default.key_verifications_per_hour_v2",
		},
		MaxQueryRangeDays: 7,
		Logger:            logging.NewNoop(),
	})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parser.Parse(context.Background(), tt.query)

			if tt.shouldPass {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}

func TestParser_ValidateTimeRange_IntervalUnits(t *testing.T) {
	tests := []struct {
		name       string
		query      string
		shouldPass bool
	}{
		{
			name:       "INTERVAL in SECONDS within retention",
			query:      "SELECT * FROM key_verifications_per_hour_v1 WHERE time >= now() - INTERVAL 86400 SECOND",
			shouldPass: true, // 1 day in seconds
		},
		{
			name:       "INTERVAL in MINUTES within retention",
			query:      "SELECT * FROM key_verifications_per_hour_v1 WHERE time >= now() - INTERVAL 2880 MINUTE",
			shouldPass: true, // 2 days in minutes
		},
		{
			name:       "INTERVAL in HOURS within retention",
			query:      "SELECT * FROM key_verifications_per_hour_v1 WHERE time >= now() - INTERVAL 72 HOUR",
			shouldPass: true, // 3 days in hours
		},
		{
			name:       "INTERVAL in DAYS within retention",
			query:      "SELECT * FROM key_verifications_per_hour_v1 WHERE time >= now() - INTERVAL 5 DAY",
			shouldPass: true,
		},
		{
			name:       "INTERVAL in WEEKS exceeds retention",
			query:      "SELECT * FROM key_verifications_per_hour_v1 WHERE time >= now() - INTERVAL 4 WEEK",
			shouldPass: false, // 28 days exceeds 7 day retention
		},
		{
			name:       "INTERVAL in MONTHS exceeds retention",
			query:      "SELECT * FROM key_verifications_per_hour_v1 WHERE time >= now() - INTERVAL 3 MONTH",
			shouldPass: false,
		},
		{
			name:       "INTERVAL in YEARS exceeds retention",
			query:      "SELECT * FROM key_verifications_per_hour_v1 WHERE time >= now() - INTERVAL 1 YEAR",
			shouldPass: false,
		},
	}

	parser := NewParser(Config{
		WorkspaceID: "test_ws",
		TableAliases: map[string]string{
			"key_verifications_per_hour_v1": "default.key_verifications_per_hour_v2",
		},
		AllowedTables: []string{
			"default.key_verifications_per_hour_v2",
		},
		MaxQueryRangeDays: 7,
		Logger:            logging.NewNoop(),
	})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parser.Parse(context.Background(), tt.query)

			if tt.shouldPass {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}

func TestParser_InjectDefaultTimeFilter(t *testing.T) {
	parser := NewParser(Config{
		WorkspaceID: "test_ws",
		TableAliases: map[string]string{
			"key_verifications_per_hour_v1": "default.key_verifications_per_hour_v2",
		},
		AllowedTables: []string{
			"default.key_verifications_per_hour_v2",
		},
		MaxQueryRangeDays: 7,
		Logger:            logging.NewNoop(),
	})

	tests := []struct {
		name          string
		query         string
		expectTimeIn  bool
		expectPattern string
	}{
		{
			name:          "query without time filter should have filter injected",
			query:         "SELECT * FROM key_verifications_per_hour_v1 WHERE key_space_id = 'ks_123'",
			expectTimeIn:  true,
			expectPattern: "time >= now() - INTERVAL 7 DAY",
		},
		{
			name:          "query with time filter should not have another injected",
			query:         "SELECT * FROM key_verifications_per_hour_v1 WHERE time >= now() - INTERVAL 5 DAY",
			expectTimeIn:  true,
			expectPattern: "time >= now() - INTERVAL 5 DAY",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parser.Parse(context.Background(), tt.query)
			require.NoError(t, err)

			if tt.expectTimeIn {
				require.Contains(t, result, "time >=", "expected time filter in result")
			}
		})
	}
}

func TestParser_ValidateTimeRange_EdgeCases(t *testing.T) {
	tests := []struct {
		name       string
		query      string
		retention  int32
		shouldPass bool
	}{
		{
			name:       "exact boundary - should pass",
			query:      "SELECT * FROM key_verifications_per_hour_v1 WHERE time >= now() - INTERVAL 7 DAY",
			retention:  7,
			shouldPass: true,
		},
		{
			name:       "one day over boundary - should fail",
			query:      "SELECT * FROM key_verifications_per_hour_v1 WHERE time >= now() - INTERVAL 8 DAY",
			retention:  7,
			shouldPass: false,
		},
		{
			name:       "addition instead of subtraction",
			query:      "SELECT * FROM key_verifications_per_hour_v1 WHERE time >= now() + INTERVAL 1 DAY",
			retention:  7,
			shouldPass: true, // Future time is always valid
		},
		{
			name:       "multiple nested functions",
			query:      "SELECT * FROM key_verifications_per_hour_v1 WHERE time >= toStartOfHour(toStartOfDay(now() - INTERVAL 5 DAY))",
			retention:  7,
			shouldPass: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewParser(Config{
				WorkspaceID: "test_ws",
				TableAliases: map[string]string{
					"key_verifications_per_hour_v1": "default.key_verifications_per_hour_v2",
				},
				AllowedTables: []string{
					"default.key_verifications_per_hour_v2",
				},
				MaxQueryRangeDays: tt.retention,
				Logger:            logging.NewNoop(),
			})

			_, err := parser.Parse(context.Background(), tt.query)

			if tt.shouldPass {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}

func TestParser_ValidateTimeRange_NumericTimestamps(t *testing.T) {
	// Unix timestamp for a date about 30 days ago (in milliseconds)
	oldTimestamp := time.Now().AddDate(0, 0, -30).UnixMilli()

	// Unix timestamp for a date about 3 days ago (in milliseconds)
	recentTimestamp := time.Now().AddDate(0, 0, -3).UnixMilli()

	tests := []struct {
		name       string
		timestamp  int64
		shouldPass bool
	}{
		{
			name:       "old unix timestamp with fromUnixTimestamp64Milli should fail",
			timestamp:  oldTimestamp,
			shouldPass: false,
		},
		{
			name:       "recent unix timestamp with fromUnixTimestamp64Milli should pass",
			timestamp:  recentTimestamp,
			shouldPass: true,
		},
	}

	parser := NewParser(Config{
		WorkspaceID: "test_ws",
		TableAliases: map[string]string{
			"key_verifications_per_hour_v1": "default.key_verifications_per_hour_v2",
		},
		AllowedTables: []string{
			"default.key_verifications_per_hour_v2",
		},
		MaxQueryRangeDays: 7,
		Logger:            logging.NewNoop(),
	})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query := "SELECT * FROM key_verifications_per_hour_v1 WHERE time >= fromUnixTimestamp64Milli(" +
				strconv.FormatInt(tt.timestamp, 10) + ")"

			_, err := parser.Parse(context.Background(), query)

			if tt.shouldPass {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}

func TestParser_ValidateTimeRange_ReversedComparisons(t *testing.T) {
	tests := []struct {
		name       string
		query      string
		shouldPass bool
	}{
		{
			name:       "reversed lower bound (value <= time) within retention",
			query:      "SELECT * FROM key_verifications_per_hour_v1 WHERE now() - INTERVAL 5 DAY <= time",
			shouldPass: true,
		},
		{
			name:       "reversed lower bound (value <= time) exceeds retention",
			query:      "SELECT * FROM key_verifications_per_hour_v1 WHERE now() - INTERVAL 90 DAY <= time",
			shouldPass: false,
		},
		{
			name:       "reversed lower bound (value < time) within retention",
			query:      "SELECT * FROM key_verifications_per_hour_v1 WHERE now() - INTERVAL 5 DAY < time",
			shouldPass: true,
		},
		{
			name:       "reversed lower bound (value < time) exceeds retention",
			query:      "SELECT * FROM key_verifications_per_hour_v1 WHERE now() - INTERVAL 90 DAY < time",
			shouldPass: false,
		},
		{
			name:       "reversed upper bound (value >= time) - should inject default filter",
			query:      "SELECT * FROM key_verifications_per_hour_v1 WHERE now() >= time",
			shouldPass: true, // Should inject default time filter
		},
		{
			name:       "reversed upper bound (value > time) - should inject default filter",
			query:      "SELECT * FROM key_verifications_per_hour_v1 WHERE now() > time",
			shouldPass: true, // Should inject default time filter
		},
		{
			name:       "combined: reversed lower bound with upper bound",
			query:      "SELECT * FROM key_verifications_per_hour_v1 WHERE now() - INTERVAL 5 DAY <= time AND time <= now()",
			shouldPass: true,
		},
		{
			name:       "combined: reversed lower bound exceeds retention with upper bound",
			query:      "SELECT * FROM key_verifications_per_hour_v1 WHERE now() - INTERVAL 90 DAY <= time AND time <= now()",
			shouldPass: false,
		},
	}

	parser := NewParser(Config{
		WorkspaceID: "test_ws",
		TableAliases: map[string]string{
			"key_verifications_per_hour_v1": "default.key_verifications_per_hour_v2",
		},
		AllowedTables: []string{
			"default.key_verifications_per_hour_v2",
		},
		MaxQueryRangeDays: 7,
		Logger:            logging.NewNoop(),
	})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parser.Parse(context.Background(), tt.query)

			if tt.shouldPass {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}

func TestParser_ValidateTimeRange_UpperBoundOnly(t *testing.T) {
	parser := NewParser(Config{
		WorkspaceID: "test_ws",
		TableAliases: map[string]string{
			"key_verifications_per_hour_v1": "default.key_verifications_per_hour_v2",
		},
		AllowedTables: []string{
			"default.key_verifications_per_hour_v2",
		},
		MaxQueryRangeDays: 7,
		Logger:            logging.NewNoop(),
	})

	tests := []struct {
		name                   string
		query                  string
		shouldInjectTimeFilter bool
	}{
		{
			name:                   "time <= old_date should inject default time filter (no lower bound)",
			query:                  "SELECT * FROM key_verifications_per_hour_v1 WHERE time <= now() - INTERVAL 90 DAY",
			shouldInjectTimeFilter: true,
		},
		{
			name:                   "time < old_date should inject default time filter (no lower bound)",
			query:                  "SELECT * FROM key_verifications_per_hour_v1 WHERE time < now() - INTERVAL 90 DAY",
			shouldInjectTimeFilter: true,
		},
		{
			name:                   "time <= now() should inject default time filter (no lower bound)",
			query:                  "SELECT * FROM key_verifications_per_hour_v1 WHERE time <= now()",
			shouldInjectTimeFilter: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parser.Parse(context.Background(), tt.query)
			require.NoError(t, err)

			if tt.shouldInjectTimeFilter {
				// The result should contain a time >= filter that was injected
				require.Contains(t, result, "time >=", "expected default time filter to be injected")
			}
		})
	}
}
