package zen

import (
	"strconv"
	"strings"
)

func selectEncoding(values []string) string {
	gzip, zstd := -1.0, -1.0
	var wildcard, identity float64
	for _, value := range values {
		for item := range strings.SplitSeq(value, ",") {
			name, quality := parseEncoding(item)
			switch strings.ToLower(name) {
			case "gzip", "x-gzip":
				gzip = max(gzip, quality)
			case "zstd":
				zstd = max(zstd, quality)
			case "identity":
				identity = max(identity, quality)
			case "*":
				wildcard = max(wildcard, quality)
			}
		}
	}

	if gzip < 0 {
		gzip = wildcard
	}
	if zstd < 0 {
		zstd = wildcard
	}
	switch {
	case zstd > identity && zstd >= gzip:
		return "zstd"
	case gzip > identity:
		return "gzip"
	default:
		return ""
	}
}

func parseEncoding(value string) (name string, quality float64) {
	name, parameter, hasParameter := strings.Cut(value, ";")
	name = strings.TrimSpace(name)
	if !hasParameter {
		return name, 1
	}
	key, weight, ok := strings.Cut(parameter, "=")
	if !ok || !strings.EqualFold(strings.TrimSpace(key), "q") {
		return name, 0
	}
	quality, err := strconv.ParseFloat(strings.TrimSpace(weight), 64)
	if err != nil || !(quality >= 0 && quality <= 1) {
		return name, 0
	}
	return name, quality
}

func hasToken(values []string, token string) bool {
	for _, value := range values {
		for item := range strings.SplitSeq(value, ",") {
			if strings.EqualFold(strings.TrimSpace(item), token) {
				return true
			}
		}
	}
	return false
}
