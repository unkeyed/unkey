package main

import (
	"fmt"
	"log"

	"github.com/unkeyed/unkey/pkg/prompt"
)

func main() {
	p := prompt.New()

	features, err := p.MultiSelect("Enable features", map[string]string{
		"log":     "Logging",
		"metrics": "Metrics",
		"trace":   "Tracing",
		"prof":    "Profiling",
	}, "log", "metrics")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Enabled: %v\n", features)
}
