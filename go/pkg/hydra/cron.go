package hydra

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// CronHandler defines the function signature for cron job handlers
type CronHandler func(ctx context.Context, payload CronPayload) error

type CronPayload struct {
	CronJobID   string `json:"cron_job_id"`
	CronName    string `json:"cron_name"`
	ScheduledAt int64  `json:"scheduled_at"`  // When this execution was scheduled
	ActualRunAt int64  `json:"actual_run_at"` // When it actually ran
	Namespace   string `json:"namespace"`
}

func (p CronPayload) Marshal() ([]byte, error) {
	return json.Marshal(p)
}

func (p *CronPayload) Unmarshal(data []byte) error {
	return json.Unmarshal(data, p)
}

func calculateNextRun(cronSpec string, from time.Time) int64 {
	schedule, err := parseCronSpec(cronSpec)
	if err != nil {
		return from.Add(5 * time.Minute).UnixMilli()
	}

	next := schedule.next(from)
	return next.UnixMilli()
}

type cronSchedule struct {
	minute uint64 // bits 0-59
	hour   uint64 // bits 0-23
	dom    uint64 // bits 1-31, day of month
	month  uint64 // bits 1-12
	dow    uint64 // bits 0-6, day of week (0=Sunday)
}

func parseCronSpec(spec string) (*cronSchedule, error) {
	fields := strings.Fields(spec)
	if len(fields) != 5 {
		return nil, errors.New("cron spec must have 5 fields")
	}

	minute, err := parseField(fields[0], 0, 59)
	if err != nil {
		return nil, fmt.Errorf("invalid minute field: %w", err)
	}

	hour, err := parseField(fields[1], 0, 23)
	if err != nil {
		return nil, fmt.Errorf("invalid hour field: %w", err)
	}

	dom, err := parseField(fields[2], 1, 31)
	if err != nil {
		return nil, fmt.Errorf("invalid day of month field: %w", err)
	}

	month, err := parseField(fields[3], 1, 12)
	if err != nil {
		return nil, fmt.Errorf("invalid month field: %w", err)
	}

	dow, err := parseField(fields[4], 0, 6)
	if err != nil {
		return nil, fmt.Errorf("invalid day of week field: %w", err)
	}

	return &cronSchedule{
		minute: minute,
		hour:   hour,
		dom:    dom,
		month:  month,
		dow:    dow,
	}, nil
}

func parseField(field string, minimum, maximum int) (uint64, error) {
	if field == "*" {
		var mask uint64
		for i := minimum; i <= maximum; i++ {
			mask |= 1 << i
		}
		return mask, nil
	}

	parts := strings.Split(field, ",")
	var mask uint64

	for _, part := range parts {
		// nolint:nestif
		if strings.Contains(part, "/") {
			stepParts := strings.Split(part, "/")
			if len(stepParts) != 2 {
				return 0, errors.New("invalid step syntax")
			}

			step, err := strconv.Atoi(stepParts[1])
			if err != nil || step <= 0 {
				return 0, errors.New("invalid step value")
			}

			rangeStart := minimum
			rangeEnd := maximum

			if stepParts[0] != "*" {
				if strings.Contains(stepParts[0], "-") {
					rangeParts := strings.Split(stepParts[0], "-")
					if len(rangeParts) != 2 {
						return 0, errors.New("invalid range syntax")
					}
					rangeStart, err = strconv.Atoi(rangeParts[0])
					if err != nil || rangeStart < minimum || rangeStart > maximum {
						return 0, errors.New("invalid range start")
					}
					rangeEnd, err = strconv.Atoi(rangeParts[1])
					if err != nil || rangeEnd < minimum || rangeEnd > maximum {
						return 0, errors.New("invalid range end")
					}
				} else {
					rangeStart, err = strconv.Atoi(stepParts[0])
					if err != nil || rangeStart < minimum || rangeStart > maximum {
						return 0, errors.New("invalid step start value")
					}
					rangeEnd = rangeStart
				}
			}

			for i := rangeStart; i <= rangeEnd; i += step {
				mask |= 1 << i
			}

		} else if strings.Contains(part, "-") {
			rangeParts := strings.Split(part, "-")
			if len(rangeParts) != 2 {
				return 0, errors.New("invalid range syntax")
			}

			start, err := strconv.Atoi(rangeParts[0])
			if err != nil || start < minimum || start > maximum {
				return 0, errors.New("invalid range start")
			}

			end, err := strconv.Atoi(rangeParts[1])
			if err != nil || end < minimum || end > maximum {
				return 0, errors.New("invalid range end")
			}

			for i := start; i <= end; i++ {
				mask |= 1 << i
			}

		} else {
			val, err := strconv.Atoi(part)
			if err != nil || val < minimum || val > maximum {
				return 0, errors.New("invalid single value")
			}
			mask |= 1 << val
		}
	}

	return mask, nil
}

func (s *cronSchedule) next(t time.Time) time.Time {
	next := t.Add(time.Minute).Truncate(time.Minute)

	end := t.Add(4 * 365 * 24 * time.Hour) // 4 years

	for next.Before(end) {
		if s.matches(next) {
			return next
		}

		next = next.Add(time.Minute)
	}

	return t.Add(365 * 24 * time.Hour)
}

func (s *cronSchedule) matches(t time.Time) bool {
	if s.minute&(1<<t.Minute()) == 0 {
		return false
	}

	if s.hour&(1<<t.Hour()) == 0 {
		return false
	}

	if s.month&(1<<t.Month()) == 0 {
		return false
	}

	domMatches := s.dom&(1<<t.Day()) != 0
	dowMatches := s.dow&(1<<int(t.Weekday())) != 0

	domIsWildcard := s.dom == s.allBits(1, 31)
	dowIsWildcard := s.dow == s.allBits(0, 6)

	if domIsWildcard && dowIsWildcard {
		return true // Both are *, so any day matches
	} else if domIsWildcard {
		return dowMatches // Only check dow
	} else if dowIsWildcard {
		return domMatches // Only check dom
	} else {
		return domMatches || dowMatches // Either can match
	}
}

func (s *cronSchedule) allBits(minimum, maximum int) uint64 {
	var mask uint64
	for i := minimum; i <= maximum; i++ {
		mask |= 1 << i
	}
	return mask
}
