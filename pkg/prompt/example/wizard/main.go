package main

import (
	"fmt"
	"log"
	"strings"

	"github.com/unkeyed/unkey/pkg/prompt"
)

func main() {
	p := prompt.New()

	wiz := p.Wizard(4)

	name, err := wiz.String("Project name", "my-project")
	if err != nil {
		log.Fatal(err)
	}

	slug := strings.ToLower(strings.ReplaceAll(name, " ", "-"))

	projectType, err := wiz.Select("Project type", map[string]string{
		"api":     "REST API",
		"web":     "Web Application",
		"cli":     "CLI Tool",
		"library": "Library",
	})
	if err != nil {
		log.Fatal(err)
	}

	features, err := wiz.MultiSelect("Enable features", map[string]string{
		"log":     "Logging",
		"metrics": "Metrics",
		"auth":    "Authentication",
		"db":      "Database",
	}, "log")
	if err != nil {
		log.Fatal(err)
	}

	confirm, err := wiz.Select("Confirm", map[string]string{
		"yes": fmt.Sprintf("Create %s (%s)", slug, projectType),
		"no":  "Cancel",
	})
	if err != nil {
		log.Fatal(err)
	}

	if confirm == "yes" {
		wiz.Done(fmt.Sprintf("Created %s with features: %v", slug, features))
	} else {
		fmt.Println("Cancelled.")
	}
}
