package main

import (
	"fmt"
	"log"

	"github.com/unkeyed/unkey/pkg/prompt"
)

func main() {
	p := prompt.New()

	date, err := p.Date("Select a date")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Selected: %s\n", date.Format("2006-01-02"))
}
