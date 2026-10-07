package cli

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestWrapText_ShortText(t *testing.T) {
	lines := wrapText("short text", 80)
	require.Equal(t, []string{"short text"}, lines)
}

func TestWrapText_ExactWidth(t *testing.T) {
	text := "exactly ten"
	lines := wrapText(text, len(text))
	require.Equal(t, []string{"exactly ten"}, lines)
}

func TestWrapText_WrapsAtWordBoundary(t *testing.T) {
	lines := wrapText("the quick brown fox jumps over the lazy dog", 20)
	require.Equal(t, []string{
		"the quick brown fox",
		"jumps over the lazy",
		"dog",
	}, lines)
}

func TestWrapText_NoSpaces(t *testing.T) {
	lines := wrapText("abcdefghij", 5)
	require.Equal(t, []string{"abcde", "fghij"}, lines)
}

func TestWrapText_EmptyString(t *testing.T) {
	lines := wrapText("", 80)
	require.Equal(t, []string{""}, lines)
}

func TestBuildFlagUsage_MutuallyExclusive(t *testing.T) {
	cmd := &Command{Flags: []Flag{
		String("request", "Request input.", Required(), MutuallyExclusive("body")),
		String("body", "Body input."),
	}}

	require.Equal(
		t,
		"Request input. (required unless --body is set; mutually exclusive with --body)",
		cmd.buildFlagUsage(cmd.Flags[0]),
	)
	require.Equal(
		t,
		"Body input. (mutually exclusive with --request)",
		cmd.buildFlagUsage(cmd.Flags[1]),
	)
}

func TestBuildFlagUsage_DoesNotExposeEnvironmentValue(t *testing.T) {
	t.Setenv("UNKEY_ROOT_KEY", "root-key-secret")
	cmd := &Command{Flags: []Flag{
		String("root-key", "Root key for authentication.", EnvVar("UNKEY_ROOT_KEY")),
	}}

	usage := cmd.buildFlagUsage(cmd.Flags[0])

	require.Contains(t, usage, "[$UNKEY_ROOT_KEY]")
	require.NotContains(t, usage, "root-key-secret")
}

func TestBuildFlagUsage_DoesNotExposeBooleanEnvironmentValue(t *testing.T) {
	t.Setenv("UNKEY_DEBUG", "true")
	cmd := &Command{Flags: []Flag{
		Bool("debug", "Enable debug mode.", EnvVar("UNKEY_DEBUG")),
	}}

	usage := cmd.buildFlagUsage(cmd.Flags[0])

	require.Contains(t, usage, "[$UNKEY_DEBUG]")
	require.NotContains(t, usage, "default: true")
}

func TestBuildFlagUsage_ShowsDeclaredDefaultInsteadOfEnvironmentValue(t *testing.T) {
	t.Setenv("UNKEY_API_URL", "https://internal.example.com")
	cmd := &Command{Flags: []Flag{
		String("api-url", "API URL.", EnvVar("UNKEY_API_URL"), Default("https://api.unkey.com")),
	}}

	usage := cmd.buildFlagUsage(cmd.Flags[0])

	require.Contains(t, usage, `default: "https://api.unkey.com"`)
	require.NotContains(t, usage, "https://internal.example.com")
}

func TestBuildFlagUsage_ShowsEmptyStringSliceDefault(t *testing.T) {
	t.Setenv("UNKEY_TAGS", "production")
	cmd := &Command{Flags: []Flag{
		StringSlice("tags", "Tags.", EnvVar("UNKEY_TAGS"), Default([]string{})),
	}}

	usage := cmd.buildFlagUsage(cmd.Flags[0])

	require.Contains(t, usage, "default: []")
	require.NotContains(t, usage, `default: [""]`)
	require.NotContains(t, usage, "production")
}

func TestBuildFlagUsage_ShowsDeclaredDefaultForEveryFlagType(t *testing.T) {
	tests := []struct {
		name         string
		environment  string
		environmentV string
		flag         func() Flag
		defaultValue string
	}{
		{
			name:         "string",
			environment:  "UNKEY_TEST_STRING",
			environmentV: "environment",
			flag:         func() Flag { return String("value", "Value.", EnvVar("UNKEY_TEST_STRING"), Default("")) },
			defaultValue: `""`,
		},
		{
			name:         "boolean",
			environment:  "UNKEY_TEST_BOOLEAN",
			environmentV: "true",
			flag:         func() Flag { return Bool("value", "Value.", EnvVar("UNKEY_TEST_BOOLEAN"), Default(false)) },
			defaultValue: "false",
		},
		{
			name:         "integer",
			environment:  "UNKEY_TEST_INTEGER",
			environmentV: "1",
			flag:         func() Flag { return Int("value", "Value.", EnvVar("UNKEY_TEST_INTEGER"), Default(0)) },
			defaultValue: "0",
		},
		{
			name:         "64-bit integer",
			environment:  "UNKEY_TEST_INT64",
			environmentV: "1",
			flag:         func() Flag { return Int64("value", "Value.", EnvVar("UNKEY_TEST_INT64"), Default(int64(0))) },
			defaultValue: "0",
		},
		{
			name:         "float",
			environment:  "UNKEY_TEST_FLOAT",
			environmentV: "1.5",
			flag:         func() Flag { return Float("value", "Value.", EnvVar("UNKEY_TEST_FLOAT"), Default(0.0)) },
			defaultValue: "0.00",
		},
		{
			name:         "string slice",
			environment:  "UNKEY_TEST_STRINGS",
			environmentV: "environment",
			flag:         func() Flag { return StringSlice("value", "Value.", EnvVar("UNKEY_TEST_STRINGS"), Default([]string{})) },
			defaultValue: "[]",
		},
		{
			name:         "duration",
			environment:  "UNKEY_TEST_DURATION",
			environmentV: "1s",
			flag: func() Flag {
				return Duration("value", "Value.", EnvVar("UNKEY_TEST_DURATION"), Default(time.Duration(0)))
			},
			defaultValue: "0s",
		},
		{
			name:         "enum",
			environment:  "UNKEY_TEST_ENUM",
			environmentV: "production",
			flag: func() Flag {
				return Enum("value", "Value.", []string{"development", "production"}, EnvVar("UNKEY_TEST_ENUM"), Default("development"))
			},
			defaultValue: `"development"`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv(test.environment, test.environmentV)
			cmd := &Command{Flags: []Flag{test.flag()}}

			usage := cmd.buildFlagUsage(cmd.Flags[0])

			require.Contains(t, usage, "default: "+test.defaultValue)
			require.NotContains(t, usage, test.environmentV)
		})
	}
}
