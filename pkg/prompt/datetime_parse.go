package prompt

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// ParseDate parses a human-readable date string into a time.Time value.
// The returned time is set to noon UTC to avoid DST edge cases.
//
// Supported formats:
//   - today, tomorrow, yesterday (case-insensitive)
//   - +1d, -2w, +3m, -1y (relative offsets: d=day, w=week, m=month, y=year)
//   - 2024-01-15 (ISO format YYYY-MM-DD)
//   - 01/15/2024 or 1/15/2024 (US format MM/DD/YYYY)
//   - 15.01.2024 (European format DD.MM.YYYY)
//
// Returns an error if the input cannot be parsed.
func ParseDate(s string) (time.Time, error) {
	s = strings.TrimSpace(strings.ToLower(s))
	if s == "" {
		return time.Time{}, fmt.Errorf("empty input")
	}

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 12, 0, 0, 0, time.UTC)

	switch s {
	case "today":
		return today, nil
	case "tomorrow":
		return today.AddDate(0, 0, 1), nil
	case "yesterday":
		return today.AddDate(0, 0, -1), nil
	}

	if len(s) >= 2 && (s[0] == '+' || s[0] == '-') {
		return parseRelativeDate(s, today)
	}

	if strings.Contains(s, "-") {
		t, err := time.Parse("2006-01-02", s)
		if err == nil {
			return time.Date(t.Year(), t.Month(), t.Day(), 12, 0, 0, 0, time.UTC), nil
		}
	}

	if strings.Contains(s, "/") {
		parts := strings.Split(s, "/")
		if len(parts) == 3 {
			month, err1 := strconv.Atoi(parts[0])
			day, err2 := strconv.Atoi(parts[1])
			year, err3 := strconv.Atoi(parts[2])
			if err1 == nil && err2 == nil && err3 == nil {
				if year < 100 {
					year += 2000
				}
				return time.Date(year, time.Month(month), day, 12, 0, 0, 0, time.UTC), nil
			}
		}
	}

	if strings.Contains(s, ".") {
		parts := strings.Split(s, ".")
		if len(parts) == 3 {
			day, err1 := strconv.Atoi(parts[0])
			month, err2 := strconv.Atoi(parts[1])
			year, err3 := strconv.Atoi(parts[2])
			if err1 == nil && err2 == nil && err3 == nil {
				if year < 100 {
					year += 2000
				}
				return time.Date(year, time.Month(month), day, 12, 0, 0, 0, time.UTC), nil
			}
		}
	}

	return time.Time{}, fmt.Errorf("invalid date format: %s", s)
}

// parseRelativeDate parses relative date offsets like +1d, -2w, +3m, -1y.
func parseRelativeDate(s string, base time.Time) (time.Time, error) {
	if len(s) < 3 {
		return time.Time{}, fmt.Errorf("invalid relative date: %s", s)
	}

	sign := 1
	if s[0] == '-' {
		sign = -1
	} else if s[0] != '+' {
		return time.Time{}, fmt.Errorf("invalid relative date: %s", s)
	}

	unit := s[len(s)-1]
	numStr := s[1 : len(s)-1]

	if numStr == "" {
		return time.Time{}, fmt.Errorf("invalid relative date: missing number in %s", s)
	}

	for _, c := range numStr {
		if c < '0' || c > '9' {
			return time.Time{}, fmt.Errorf("invalid relative date number: %s", s)
		}
	}

	num, err := strconv.Atoi(numStr)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid relative date number: %s", s)
	}

	num *= sign

	switch unit {
	case 'd':
		return base.AddDate(0, 0, num), nil
	case 'w':
		return base.AddDate(0, 0, num*7), nil
	case 'm':
		return base.AddDate(0, num, 0), nil
	case 'y':
		return base.AddDate(num, 0, 0), nil
	default:
		return time.Time{}, fmt.Errorf("invalid relative date unit: %c (use d, w, m, or y)", unit)
	}
}

// ParseTime parses a human-readable time string into hour and minute values.
//
// Supported formats:
//   - now (current time)
//   - 14:30, 14:30:00 (24-hour format HH:MM or HH:MM:SS)
//   - 2:30pm, 2:30 PM, 2:30PM (12-hour format with am/pm)
//   - 1430 (military time HHMM)
//
// Returns hour (0-23) and minute (0-59), or an error if parsing fails.
func ParseTime(s string) (int, int, error) {
	s = strings.TrimSpace(strings.ToLower(s))
	if s == "" {
		return 0, 0, fmt.Errorf("empty input")
	}

	if s == "now" {
		now := time.Now()
		return now.Hour(), now.Minute(), nil
	}

	isPM := false
	isAM := false

	if strings.HasSuffix(s, "pm") {
		isPM = true
		s = strings.TrimSuffix(s, "pm")
		s = strings.TrimSpace(s)
	} else if strings.HasSuffix(s, "am") {
		isAM = true
		s = strings.TrimSuffix(s, "am")
		s = strings.TrimSpace(s)
	} else if strings.HasSuffix(s, "p") {
		isPM = true
		s = strings.TrimSuffix(s, "p")
		s = strings.TrimSpace(s)
	} else if strings.HasSuffix(s, "a") {
		isAM = true
		s = strings.TrimSuffix(s, "a")
		s = strings.TrimSpace(s)
	}

	if strings.Contains(s, ":") {
		parts := strings.Split(s, ":")
		if len(parts) >= 2 {
			hour, err1 := strconv.Atoi(parts[0])
			minute, err2 := strconv.Atoi(parts[1])
			if err1 == nil && err2 == nil {
				hour = convert12To24(hour, isPM, isAM)
				if hour < 0 || hour > 23 || minute < 0 || minute > 59 {
					return 0, 0, fmt.Errorf("invalid time: hour must be 0-23, minute must be 0-59")
				}
				return hour, minute, nil
			}
		}
	}

	if len(s) == 4 {
		hour, err1 := strconv.Atoi(s[:2])
		minute, err2 := strconv.Atoi(s[2:])
		if err1 == nil && err2 == nil {
			hour = convert12To24(hour, isPM, isAM)
			if hour >= 0 && hour <= 23 && minute >= 0 && minute <= 59 {
				return hour, minute, nil
			}
		}
	}

	if len(s) == 3 {
		hour, err1 := strconv.Atoi(s[:1])
		minute, err2 := strconv.Atoi(s[1:])
		if err1 == nil && err2 == nil {
			hour = convert12To24(hour, isPM, isAM)
			if hour >= 0 && hour <= 23 && minute >= 0 && minute <= 59 {
				return hour, minute, nil
			}
		}
	}

	if hour, err := strconv.Atoi(s); err == nil {
		hour = convert12To24(hour, isPM, isAM)
		if hour >= 0 && hour <= 23 {
			return hour, 0, nil
		}
	}

	return 0, 0, fmt.Errorf("invalid time format: %s", s)
}

// convert12To24 converts a 12-hour time to 24-hour format based on AM/PM indicators.
func convert12To24(hour int, isPM, isAM bool) int {
	if isPM && hour < 12 {
		hour += 12
	} else if isAM && hour == 12 {
		hour = 0
	}
	return hour
}

// ParseDateTime parses a combined date and time string.
// The date and time parts should be separated by a space or 'T' (ISO 8601).
//
// Supported formats:
//   - 2024-01-15 14:30
//   - 2024-01-15T14:30
//   - today 2pm
//   - tomorrow 14:30
//   - +1d 9:00am
//
// Returns the parsed time, or an error if parsing fails.
func ParseDateTime(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, fmt.Errorf("empty input")
	}

	parts := strings.Fields(s)
	if len(parts) == 0 {
		return time.Time{}, fmt.Errorf("empty input")
	}

	if len(parts) == 2 && (parts[0][len(parts[0])-1] == 'T' || parts[0][len(parts[0])-1] == 't') {
		parts[0] = parts[0][:len(parts[0])-1]
	}

	if len(parts) == 1 && strings.Contains(parts[0], "T") {
		idx := strings.Index(parts[0], "T")
		parts = []string{parts[0][:idx], parts[0][idx+1:]}
	} else if len(parts) == 1 && strings.Contains(strings.ToUpper(parts[0]), "T") && !strings.HasPrefix(strings.ToLower(parts[0]), "today") && !strings.HasPrefix(strings.ToLower(parts[0]), "tomorrow") {
		lower := strings.ToLower(parts[0])
		idx := strings.Index(lower, "t")
		if idx > 0 && idx < len(parts[0])-1 {
			parts = []string{parts[0][:idx], parts[0][idx+1:]}
		}
	}

	if len(parts) == 1 {
		if dt, err := ParseDate(parts[0]); err == nil {
			return dt, nil
		}
		if h, m, err := ParseTime(parts[0]); err == nil {
			now := time.Now()
			return time.Date(now.Year(), now.Month(), now.Day(), h, m, 0, 0, time.UTC), nil
		}
		return time.Time{}, fmt.Errorf("invalid datetime format: %s", s)
	}

	dateStr := parts[0]
	timeStr := strings.Join(parts[1:], " ")

	dt, err := ParseDate(dateStr)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid date in datetime: %w", err)
	}

	hour, minute, err := ParseTime(timeStr)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid time in datetime: %w", err)
	}

	return time.Date(dt.Year(), dt.Month(), dt.Day(), hour, minute, 0, 0, time.UTC), nil
}
