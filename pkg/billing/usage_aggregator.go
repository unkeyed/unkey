package billing

import (
	"context"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/unkeyed/unkey/pkg/clickhouse"
	"github.com/unkeyed/unkey/pkg/fault"
)

// extractString extracts a string from various types returned by ClickHouse queries
func extractString(value interface{}) (string, bool) {
	if value == nil {
		return "", false
	}

	switch v := value.(type) {
	case string:
		return v, true
	case []byte:
		return string(v), true
	default:
		return extractStringFromReflect(value)
	}
}

// extractStringFromReflect handles non-string types using reflection
func extractStringFromReflect(value interface{}) (string, bool) {
	if value == nil {
		return "", false
	}

	rv := reflect.ValueOf(value)
	if rv.Kind() == reflect.Interface {
		rv = rv.Elem()
	}
	if rv.Kind() == reflect.Ptr {
		rv = rv.Elem()
	}

	// Handle struct types (Variant-like)
	if rv.Kind() == reflect.Struct {
		// Try to get the Value field first
		fv := rv.FieldByName("Value")
		if fv.IsValid() && fv.Kind() == reflect.Interface {
			inner := fv.Interface()
			if s, ok := inner.(string); ok {
				return s, true
			}
		}

		// If Value field not found or not a string, try all fields
		for i := 0; i < rv.NumField(); i++ {
			field := rv.Field(i)
			if field.Kind() == reflect.Interface {
				inner := field.Interface()
				if s, ok := inner.(string); ok {
					return s, true
				}
			}
		}

		// Try parsing the string representation
		strValue := fmt.Sprintf("%v", value)
		// The format is like "{user_123}" so extract the content
		if strings.HasPrefix(strValue, "{") && strings.HasSuffix(strValue, "}") {
			content := strValue[1 : len(strValue)-1]
			if content != "" {
				return content, true
			}
		}
	}

	return "", false
}

// extractInt64 extracts an int64 from various types returned by ClickHouse queries
func extractInt64(value interface{}) (int64, bool) {
	if value == nil {
		return 0, false
	}

	switch v := value.(type) {
	case int64:
		return v, true
	case int:
		return int64(v), true
	case uint64:
		return int64(v), true
	case int32:
		return int64(v), true
	case uint32:
		return int64(v), true
	default:
		return extractInt64FromReflect(value)
	}
}

// extractInt64FromReflect handles non-int64 types using reflection
func extractInt64FromReflect(value interface{}) (int64, bool) {
	if value == nil {
		return 0, false
	}

	rv := reflect.ValueOf(value)
	if rv.Kind() == reflect.Interface {
		rv = rv.Elem()
	}
	if rv.Kind() == reflect.Ptr {
		rv = rv.Elem()
	}

	if rv.Kind() == reflect.Struct {
		// Try to get the Value field first
		fv := rv.FieldByName("Value")
		if fv.IsValid() && fv.Kind() == reflect.Interface {
			inner := fv.Interface()
			switch inner.(type) {
			case int64:
				return inner.(int64), true
			case int:
				return int64(inner.(int)), true
			case uint64:
				return int64(inner.(uint64)), true
			}
		}

		// If Value field not found or not an int, try all fields
		for i := 0; i < rv.NumField(); i++ {
			field := rv.Field(i)
			if field.Kind() == reflect.Interface {
				inner := field.Interface()
				switch inner.(type) {
				case int64:
					return inner.(int64), true
				case int:
					return int64(inner.(int)), true
				case uint64:
					return int64(inner.(uint64)), true
				}
			}
		}

		// Try parsing the string representation
		strValue := fmt.Sprintf("%v", value)
		// The format is like "{15}" so extract the content
		if strings.HasPrefix(strValue, "{") && strings.HasSuffix(strValue, "}") {
			content := strValue[1 : len(strValue)-1]
			if val, err := strconv.ParseInt(content, 10, 64); err == nil {
				return val, true
			}
		}
	}

	return 0, false
}

// Usage is defined in end_user.go to avoid duplication

// UsageAggregator aggregates usage data from ClickHouse for billing purposes.
// It queries verification and rate limit events, filtering by external_id to
// associate usage with specific end users.
type UsageAggregator interface {
	// AggregateUsage aggregates usage from ClickHouse for a billing period.
	// Returns a map of external_id to Usage for all end users with usage in the period.
	// Events without external_id are excluded from results.
	AggregateUsage(ctx context.Context, workspaceID string, periodStart, periodEnd time.Time) (map[string]*Usage, error)

	// GetEndUserUsage gets usage for a specific end user in a billing period.
	// Returns zero usage if no events found for the external_id.
	GetEndUserUsage(ctx context.Context, workspaceID, externalID string, periodStart, periodEnd time.Time) (*Usage, error)
}

type usageAggregator struct {
	clickhouse clickhouse.Querier
}

// NewUsageAggregator creates a new UsageAggregator that queries ClickHouse.
func NewUsageAggregator(ch clickhouse.Querier) UsageAggregator {
	return &usageAggregator{
		clickhouse: ch,
	}
}

// AggregateUsage aggregates usage from ClickHouse for a billing period.
// It queries end_user_billable_verifications_per_month_v1 table,
// filtering by workspace_id, year, and month.
// Results are grouped by external_id.
//
// Requirements: 4.1, 4.2, 4.3, 4.4, 4.6
func (u *usageAggregator) AggregateUsage(ctx context.Context, workspaceID string, periodStart, periodEnd time.Time) (map[string]*Usage, error) {
	// Build list of months to query
	months := []struct {
		year  int16
		month int8
	}{}

	current := periodStart.Truncate(24 * time.Hour)
	for !current.After(periodEnd) {
		months = append(months, struct {
			year  int16
			month int8
		}{
			year:  int16(current.Year()),
			month: int8(current.Month()),
		})
		current = current.AddDate(0, 1, 0)
	}

	// Build usage map
	usageMap := make(map[string]*Usage)

	// Query verifications for all months in the period
	verificationsQuery := `
		SELECT
			CAST(external_id AS String) as external_id,
			CAST(sum(count) AS Int64) as count
		FROM default.end_user_billable_verifications_per_month_v1
		WHERE workspace_id = ?
			AND external_id != ''
			AND year >= ? AND year <= ?
			AND (
				(year > ? AND year < ?) OR
				(year = ? AND month >= ?) OR
				(year = ? AND month <= ?)
			)
		GROUP BY external_id
	`

	startYear := int16(periodStart.Year())
	startMonth := int8(periodStart.Month())
	endYear := int16(periodEnd.Year())
	endMonth := int8(periodEnd.Month())

	fmt.Printf("DEBUG: AggregateUsage: workspaceID=%s, startYear=%d, startMonth=%d, endYear=%d, endMonth=%d\n",
		workspaceID, startYear, startMonth, endYear, endMonth)

	verificationRows, err := u.clickhouse.QueryToMaps(ctx, verificationsQuery,
		workspaceID,
		startYear, endYear,   // year >= startYear AND year <= endYear
		startYear, endYear,    // (year > startYear AND year < endYear) - middle months
		startYear, startMonth, // (year = startYear AND month >= startMonth) - start month
		endYear, endMonth)     // (year = endYear AND month <= endMonth) - end month
	if err != nil {
		return nil, fault.Wrap(err, fault.Internal("failed to query verifications for usage aggregation"))
	}

	fmt.Printf("DEBUG: AggregateUsage: verificationRows=%d\n", len(verificationRows))
	for i, row := range verificationRows {
		fmt.Printf("DEBUG: AggregateUsage: verificationRow[%d]=%v\n", i, row)
	}

	// Query credits for all months in the period
	creditsQuery := `
		SELECT
			CAST(external_id AS String) as external_id,
			CAST(sum(count) AS Int64) as count
		FROM default.end_user_billable_credits_per_month_v1
		WHERE workspace_id = ?
			AND external_id != ''
			AND year >= ? AND year <= ?
			AND (
				(year > ? AND year < ?) OR
				(year = ? AND month >= ?) OR
				(year = ? AND month <= ?)
			)
		GROUP BY external_id
	`

	creditRows, err := u.clickhouse.QueryToMaps(ctx, creditsQuery,
		workspaceID,
		startYear, endYear,    // year >= startYear AND year <= endYear
		startYear, endYear,     // middle months
		startYear, startMonth,  // start month
		endYear, endMonth)      // end month
	if err != nil {
		return nil, fault.Wrap(err, fault.Internal("failed to query credits for usage aggregation"))
	}

	fmt.Printf("DEBUG: AggregateUsage: verificationRows=%d, creditRows=%d\n", len(verificationRows), len(creditRows))

	// Process verifications
	for _, row := range verificationRows {
		rawExternalID := row["external_id"]
		externalID, ok := extractString(rawExternalID)
		fmt.Printf("DEBUG: AggregateUsage: externalID=%q, ok=%v\n", externalID, ok)
		if !ok || externalID == "" {
			fmt.Printf("DEBUG: AggregateUsage: skipping row with invalid external_id\n")
			continue
		}

		rawCount := row["count"]
		count, ok := extractInt64(rawCount)
		fmt.Printf("DEBUG: AggregateUsage: rawCount type=%T, value=%v, count=%d, ok=%v\n", rawCount, rawCount, count, ok)
		if !ok {
			fmt.Printf("DEBUG: AggregateUsage: skipping row with invalid count for external_id=%s\n", externalID)
			continue
		}

		fmt.Printf("DEBUG: AggregateUsage: processing verification row: external_id=%s, count=%d\n", externalID, count)
		if _, exists := usageMap[externalID]; !exists {
			fmt.Printf("DEBUG: AggregateUsage: creating new Usage for external_id=%s\n", externalID)
			usageMap[externalID] = &Usage{
				ExternalID:     externalID,
				Verifications:  0,
				KeysWithAccess: 0,
				Credits:        0,
			}
		} else {
			fmt.Printf("DEBUG: AggregateUsage: updating existing Usage for external_id=%s\n", externalID)
		}
		usageMap[externalID].Verifications = count
		fmt.Printf("DEBUG: AggregateUsage: usageMap[%s].Verifications = %d\n", externalID, usageMap[externalID].Verifications)
	}

	// Process credits
	for _, row := range creditRows {
		rawExternalID := row["external_id"]
		externalID, ok := extractString(rawExternalID)
		if !ok || externalID == "" {
			continue
		}

		rawCount := row["count"]
		count, ok := extractInt64(rawCount)
		if !ok {
			continue
		}

		if _, exists := usageMap[externalID]; !exists {
			usageMap[externalID] = &Usage{
				ExternalID:     externalID,
				Verifications:  0,
				KeysWithAccess: 0,
				Credits:        0,
			}
		}
		usageMap[externalID].Credits = count
	}

	fmt.Printf("DEBUG: AggregateUsage: before return: usageMap len=%d\n", len(usageMap))
	for k, v := range usageMap {
		fmt.Printf("DEBUG: AggregateUsage: usageMap[%s] = {Verifications: %d, Credits: %d}\n", k, v.Verifications, v.Credits)
	}

	return usageMap, nil
}

// GetEndUserUsage gets usage for a specific end user in a billing period.
// Returns zero usage if no events found for the external_id.
//
// Requirements: 4.1, 4.2, 4.3, 4.4, 4.6
func (u *usageAggregator) GetEndUserUsage(ctx context.Context, workspaceID, externalID string, periodStart, periodEnd time.Time) (*Usage, error) {
	// Get year and month from the period
	year := int16(periodStart.Year())
	month := int8(periodStart.Month())

	usage := &Usage{
		ExternalID:     externalID,
		Verifications:  0,
		KeysWithAccess: 0,
		Credits:        0,
	}

	// Query verifications from the end-user billable aggregated table
	verificationsQuery := `
		SELECT CAST(sum(count) AS Int64) as count
		FROM default.end_user_billable_verifications_per_month_v1
		WHERE workspace_id = ?
			AND external_id = ?
			AND year = ?
			AND month = ?
	`

	var verificationCount int64
	err := u.clickhouse.Conn().QueryRow(ctx, verificationsQuery, workspaceID, externalID, year, month).Scan(&verificationCount)
	if err != nil && err.Error() != "sql: no rows in result set" {
		return nil, fault.Wrap(err, fault.Internal("failed to query verifications for end user"))
	}
	usage.Verifications = verificationCount

	// Query credits from the end-user billable aggregated table
	creditsQuery := `
		SELECT CAST(sum(count) AS Int64) as count
		FROM default.end_user_billable_credits_per_month_v1
		WHERE workspace_id = ?
			AND external_id = ?
			AND year = ?
			AND month = ?
	`

	var creditsCount int64
	err = u.clickhouse.Conn().QueryRow(ctx, creditsQuery, workspaceID, externalID, year, month).Scan(&creditsCount)
	if err != nil && err.Error() != "sql: no rows in result set" {
		return nil, fault.Wrap(err, fault.Internal("failed to query credits for end user"))
	}
	usage.Credits = creditsCount

	return usage, nil
}
