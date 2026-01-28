package main

import (
	"fmt"
	"log"

	"github.com/unkeyed/unkey/pkg/prompt"
)

func main() {
	p := prompt.New()

	// String with default
	name, err := p.String("Enter your name", "Anonymous")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Hello, %s!\n\n", name)

	// Int with default
	age, err := p.Int("Enter your age", 25)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("You are %d years old.\n\n", age)

	// Select with default (staging pre-selected)
	env, err := p.Select("Select environment", map[string]string{
		"dev":  "Development",
		"stg":  "Staging",
		"prod": "Production",
	}, "stg")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Selected: %s\n\n", env) // e.g. "stg"

	// MultiSelect with defaults (log and metrics pre-selected)
	features, err := p.MultiSelect("Enable features", map[string]string{
		"log":     "Logging",
		"metrics": "Metrics",
		"trace":   "Tracing",
		"prof":    "Profiling",
	}, "log", "metrics")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Enabled: %v\n\n", features)

	// Date picker with calendar navigation
	date, err := p.Date("Select a date")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Selected date: %s\n\n", date.Format("2006-01-02"))

	// Time picker with 5-minute increments
	selectedTime, err := p.Time("Select a time", 5)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Selected time: %s\n\n", selectedTime.Format("15:04"))

	// DateTime picker combining both
	dateTime, err := p.DateTime("Select date and time", 15)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Selected datetime: %s\n", dateTime.Format("2006-01-02 15:04"))
}
