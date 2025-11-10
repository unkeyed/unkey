//go:build ignore

package main

import (
	"bytes"
	"fmt"
	"os"
	"regexp"
)

func main() {
	data, err := os.ReadFile("gen_easyjson.go")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to read gen_easyjson.go: %v\n", err)
		os.Exit(1)
	}

	// Fix nullable.Nullable fields that call UnmarshalJSON
	// Pattern: if in.IsNull() { in.Skip() } else { if data := in.Raw(); in.Ok() { in.AddError((out.X).UnmarshalJSON(data)) } }
	// Replace: if data := in.Raw(); in.Ok() { in.AddError((out.X).UnmarshalJSON(data)) }

	pattern := regexp.MustCompile(`(?s)if in\.IsNull\(\) \{\s+in\.Skip\(\)\s+\} else \{\s+(if data := in\.Raw\(\); in\.Ok\(\) \{\s+in\.AddError\(\([^)]+\)\.UnmarshalJSON\(data\)\)\s+\}\s+)\}`)

	fixed := pattern.ReplaceAll(data, []byte("$1"))

	if bytes.Equal(data, fixed) {
		fmt.Println("No nullable fixes needed")
	} else {
		if err := os.WriteFile("gen_easyjson.go", fixed, 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to write gen_easyjson.go: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Fixed nullable.Nullable fields in gen_easyjson.go")
	}
}
