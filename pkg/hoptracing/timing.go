package hoptracing

import (
	"fmt"
	"strconv"
	"strings"
)

type TimingMetric struct {
	Kind       HopKind
	Region     string
	ID         string
	DurationMs int64
}

func (t TimingMetric) String() string {
	return fmt.Sprintf("unkey_%s;desc=\"%s/%s\";dur=%d", t.Kind, t.Region, t.ID, t.DurationMs)
}

func parseTiming(s string) []TimingMetric {
	if s == "" {
		return nil
	}
	return nil
}

func parseTimingMetric(s string) (TimingMetric, bool) {
	empty := TimingMetric{Kind: "", Region: "", ID: "", DurationMs: 0}

	kindEnd := strings.Index(s, ";")
	if kindEnd < 0 || !strings.HasPrefix(s, "unkey_") {
		return empty, false
	}
	kindStr := s[6:kindEnd]

	descStart := strings.Index(s, "desc=\"")
	if descStart < 0 {
		return empty, false
	}
	descEnd := strings.Index(s[descStart+6:], "\"")
	if descEnd < 0 {
		return empty, false
	}
	desc := s[descStart+6 : descStart+6+descEnd]

	durStart := strings.Index(s, "dur=")
	if durStart < 0 {
		return empty, false
	}
	durStr := s[durStart+4:]
	if commaIdx := strings.Index(durStr, ","); commaIdx > 0 {
		durStr = durStr[:commaIdx]
	}
	dur, err := strconv.ParseInt(durStr, 10, 64)
	if err != nil {
		return empty, false
	}

	slashIdx := strings.Index(desc, "/")
	if slashIdx < 0 {
		return empty, false
	}

	return TimingMetric{
		Kind:       HopKind(kindStr),
		Region:     desc[:slashIdx],
		ID:         desc[slashIdx+1:],
		DurationMs: dur,
	}, true
}
