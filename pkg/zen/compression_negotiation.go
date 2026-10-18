package zen

import (
	"strconv"
	"strings"
)

func selectEncoding(values []string) string {
	qualities := make(map[string]float64, 4)
	for _, value := range values {
		for _, item := range strings.Split(value, ",") {
			name, parameter, hasParameter := strings.Cut(item, ";")
			name = strings.ToLower(strings.TrimSpace(name))
			if name == "x-gzip" {
				name = "gzip"
			}
			quality := 1.0
			if hasParameter {
				key, weight, ok := strings.Cut(parameter, "=")
				parsed, err := strconv.ParseFloat(strings.TrimSpace(weight), 64)
				if !ok || !strings.EqualFold(strings.TrimSpace(key), "q") || err != nil || !(parsed >= 0 && parsed <= 1) {
					quality = 0
				} else {
					quality = parsed
				}
			}
			qualities[name] = max(qualities[name], quality)
		}
	}

	selected := ""
	bestQuality := qualities["identity"]
	for _, encoding := range []string{"zstd", "gzip"} {
		quality, explicit := qualities[encoding]
		if !explicit {
			quality = qualities["*"]
		}
		if quality > bestQuality {
			selected = encoding
			bestQuality = quality
		}
	}
	return selected
}

func hasToken(values []string, token string) bool {
	for _, value := range values {
		for _, item := range strings.Split(value, ",") {
			if strings.EqualFold(strings.TrimSpace(item), token) {
				return true
			}
		}
	}
	return false
}
