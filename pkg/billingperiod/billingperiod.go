// Package billingperiod parses and works with the "YYYY-MM" billing period
// keys used to key the monthly cron handlers (quota check, deploy billing
// push, ...). Centralised so the handlers share one parser instead of each
// reimplementing the same string splitting.
package billingperiod

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Period is a calendar month: the unit a workspace is billed over and the
// Restate object key ("YYYY-MM") that serializes a month's cron runs.
type Period struct {
	Year  int
	Month time.Month
}

// Parse reads a "YYYY-MM" key into a Period.
func Parse(key string) (Period, error) {
	parts := strings.Split(key, "-")
	// Enforce the exact YYYY-MM widths so malformed keys like "2026-5" or
	// "26-05" are rejected rather than silently parsed into the wrong month.
	if len(parts) != 2 || len(parts[0]) != 4 || len(parts[1]) != 2 {
		return Period{}, fmt.Errorf("expected YYYY-MM format, got %q", key)
	}

	year, err := strconv.Atoi(parts[0])
	if err != nil {
		return Period{}, fmt.Errorf("invalid year: %w", err)
	}

	month, err := strconv.Atoi(parts[1])
	if err != nil {
		return Period{}, fmt.Errorf("invalid month: %w", err)
	}
	if month < 1 || month > 12 {
		return Period{}, fmt.Errorf("month must be 1-12, got %d", month)
	}

	return Period{Year: year, Month: time.Month(month)}, nil
}

// Start is midnight UTC on the first day of the month.
func (p Period) Start() time.Time {
	return time.Date(p.Year, p.Month, 1, 0, 0, 0, 0, time.UTC)
}

// End is midnight UTC on the first day of the following month: the exclusive
// upper bound of the period, and the instant the renewal invoice is created
// for subscriptions anchored at midnight on the 1st.
func (p Period) End() time.Time {
	return p.Start().AddDate(0, 1, 0)
}
