package main

import (
	"fmt"
	"log"

	"github.com/unkeyed/unkey/pkg/prompt"
)

func main() {
	p := prompt.New()

	env, err := p.Select("Select environment", map[string]string{
		"dev":  "Development",
		"stg":  "Staging",
		"prod": "Production",
	}, "stg")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Selected: %s\n", env)
}
