package main

import (
	"fmt"
	"log"

	"github.com/unkeyed/unkey/pkg/prompt"
)

func main() {
	p := prompt.New()

	dt, err := p.DateTime("Select date and time", 15)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Selected: %s\n", dt.Format("2006-01-02 15:04"))
}
