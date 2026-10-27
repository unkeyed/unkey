package main

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
)

func TestUsesOutputFormat(t *testing.T) {
	tests := []struct {
		name     string
		owner    string
		repo     string
		tag      string
		sha      string
		expected string
	}{
		{
			name:     "standard action",
			owner:    "actions",
			repo:     "checkout",
			tag:      "v5.0.0",
			sha:      "abc1234",
			expected: "- uses: actions/checkout@abc1234 # v5.0.0\n",
		},
		{
			name:     "custom owner",
			owner:    "docker",
			repo:     "setup-buildx-action",
			tag:      "v3.11.1",
			sha:      "e468171",
			expected: "- uses: docker/setup-buildx-action@e468171 # v3.11.1\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			old := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			// Call the output function
			err := outputResult(tt.owner, tt.repo, tt.tag, tt.sha)

			w.Close()
			os.Stdout = old

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			var buf bytes.Buffer
			io.Copy(&buf, r)
			got := buf.String()

			if got != tt.expected {
				t.Errorf("got %q, want %q", got, tt.expected)
			}

			if !strings.HasPrefix(got, "- uses: ") {
				t.Error("output should start with '- uses: '")
			}
			if !strings.Contains(got, "@") {
				t.Error("output should contain '@' separator")
			}
			if !strings.Contains(got, " # ") {
				t.Error("output should contain ' # ' comment separator")
			}
		})
	}
}
