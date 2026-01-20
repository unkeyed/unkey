package hoptracing

import (
	"fmt"
	"strings"
)

type HopKind string

const (
	HopFrontline HopKind = "fl"
	HopSentinel  HopKind = "se"
	HopInstance  HopKind = "in"
)

const MaxHops = 8

type Hop struct {
	Kind   HopKind
	Region string
	ID     string
}

func (h Hop) String() string {
	return fmt.Sprintf("%s@%s/%s", h.Kind, h.Region, h.ID)
}

func parseRoute(s string) []Hop {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ", ")
	hops := make([]Hop, 0, len(parts))
	for _, p := range parts {
		if h, ok := parseHop(p); ok {
			hops = append(hops, h)
		}
	}
	return hops
}

func parseHop(s string) (Hop, bool) {
	atIdx := strings.Index(s, "@")
	slashIdx := strings.Index(s, "/")
	if atIdx < 1 || slashIdx <= atIdx {
		return Hop{Kind: "", Region: "", ID: ""}, false
	}
	return Hop{
		Kind:   HopKind(s[:atIdx]),
		Region: s[atIdx+1 : slashIdx],
		ID:     s[slashIdx+1:],
	}, true
}
