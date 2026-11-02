package api

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/cmd/api/internal/guidance"
)

func TestEveryResourceHasGuidance(t *testing.T) {
	for _, group := range groups() {
		for _, command := range group.Commands {
			legacy := group.Name + " " + command.Name
			schema, ok := guidance.Lookup(legacy)
			require.True(t, ok, "generate guidance for %s", legacy)
			require.Nil(t, command.Prepare, "legacy commands must not prompt")
			for _, flag := range command.Flags {
				switch flag.Name() {
				case "body", "root-key", "api-url", "config", "output", "json", "debug", "yes", "error-format", "reveal":
					continue
				}
				if schema.Type == "array" {
					require.Equal(t, "limits", flag.Name())
					continue
				}
				if legacy == "gateway update-policy" && flag.Name() == "policy" {
					for _, field := range []string{"name", "enabled", "match", "keyauth", "ratelimit", "firewall", "openapi", "logging"} {
						require.Contains(t, schema.Properties, field)
					}
					continue
				}
				found := false
				for name := range schema.Properties {
					if strings.EqualFold(name, strings.ReplaceAll(flag.Name(), "-", "")) {
						found = true
						break
					}
				}
				require.True(t, found, "%s --%s has no guidance field", legacy, flag.Name())
			}
			output, _, err := runCLI(t, append(strings.Fields(resourcePaths[legacy]), "--help")...)
			require.NoError(t, err)
			require.Contains(t, output, "--interactive")
			require.Contains(t, output, "--no-input")
		}
	}
}

func TestGuidanceDoesNotPromptInScripts(t *testing.T) {
	for _, args := range [][]string{
		{"projects", "create"},
		{"projects", "create", "--no-input"},
		{"projects", "create", "--interactive=false"},
		{"keys", "create", "--no-input"},
		{"projects", "create", "--interactive"},
		{"projects", "create", "--interactive", "--no-input"},
		{"projects", "create", "--interactive", "--body="},
	} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			out, diagnostic, err := runCLI(t, args...)
			require.Error(t, err)
			require.Empty(t, out)
			require.Empty(t, diagnostic)
		})
	}
}
