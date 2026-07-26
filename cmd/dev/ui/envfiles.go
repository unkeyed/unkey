package ui

import (
	"os"
	"strings"
)

// Dev env files loaded for local tooling, same sources as mise/Tilt.
var devEnvFiles = []string{
	".env",
	"dev/.env",
	"dev/.env.stripe",
	"dev/.env.seed",
}

func loadDevEnvFiles() {
	for _, path := range devEnvFiles {
		_ = loadEnvFile(path)
	}
}

// loadEnvFile sets KEY=VALUE pairs from a dotenv file. Existing process env wins.
func loadEnvFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	for line := range strings.SplitSeq(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if after, ok := strings.CutPrefix(line, "export "); ok {
			line = strings.TrimSpace(after)
		}
		key, val, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		val = strings.TrimSpace(val)
		val = strings.Trim(val, `"'`)
		if key == "" || os.Getenv(key) != "" {
			continue
		}
		_ = os.Setenv(key, val)
	}
	return nil
}
