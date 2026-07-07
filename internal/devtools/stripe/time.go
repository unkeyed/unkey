package stripe

import "time"

// FormatTime renders a unix timestamp in RFC3339 UTC.
func FormatTime(unixSeconds int64) string {
	return time.Unix(unixSeconds, 0).UTC().Format(time.RFC3339)
}
