package util

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/unkeyed/unkey/pkg/cli"
)

// Output prints the API response to stdout. With -o json, it prints the full
// response envelope as-is for piping. Otherwise it prints the request ID with
// latency, followed by the data payload as indented JSON.
func Output(cmd *cli.Command, v any, latency time.Duration) error {
	if cmd.String("output") == "json" {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(v)
	}

	raw, err := json.Marshal(v)
	if err != nil {
		return err
	}

	var envelope struct {
		Meta struct {
			RequestID string `json:"requestId"`
		} `json:"meta"`
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(v)
	}

	if envelope.Meta.RequestID != "" {
		fmt.Printf("%s (took %s)\n\n", envelope.Meta.RequestID, latency.Round(time.Millisecond))
	}

	indented, err := json.MarshalIndent(json.RawMessage(envelope.Data), "", "  ")
	if err != nil {
		if _, wErr := os.Stdout.Write(envelope.Data); wErr != nil {
			return wErr
		}
		_, wErr := fmt.Fprintln(os.Stdout)
		return wErr
	}
	if _, wErr := os.Stdout.Write(indented); wErr != nil {
		return wErr
	}
	_, wErr := fmt.Fprintln(os.Stdout)
	return wErr
}
