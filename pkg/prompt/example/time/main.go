package main

import (
	"fmt"
	"log"

	"github.com/unkeyed/unkey/pkg/prompt"
)

func main() {
	p := prompt.New()

	t, err := p.Time("Select a time", 5)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Selected: %s\n", t.Format("15:04"))
}
