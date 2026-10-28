package main

import (
	"go/scanner"
	"go/token"
	"testing"

	"github.com/stretchr/testify/require"
)

const sourceHeader = "package sample\n\nimport \"github.com/unkeyed/unkey/pkg/cli\"\n\n"

func TestFormatCommands(t *testing.T) {
	help := "`Delete an app.\n\n  Keep indentation, {braces}, and commas.\n\tKeep tabs too.\n`"
	tests := []struct {
		name   string
		source string
		want   string
	}{
		{
			name: "multiline_raw_help",
			source: `func command() *cli.Command {
	return &cli.Command{Name: "delete-app", Description: ` + help + ` + disclaimer, Examples: []string{"first", "second"}, Flags: []cli.Flag{cli.String("body", "JSON", cli.Required()), rootFlag()}, Action: func() error {
		return nil
	}}
}
`,
			want: `func command() *cli.Command {
	return &cli.Command{
		Name: "delete-app",
		Description: ` + help + ` + disclaimer,
		Examples: []string{
			"first",
			"second",
		},
		Flags: []cli.Flag{
			cli.String("body", "JSON", cli.Required()),
			rootFlag(),
		},
		Action: func() error {
			return nil
		},
	}
}
`,
		},
		{
			name: "nested_commands_leave_unrelated_literals_and_calls_compact",
			source: `var command = &cli.Command{Name: "root", Commands: []*cli.Command{{Name: "child", Examples: []string{"child"}, Commands: []*cli.Command{leaf(), &cli.Command{Name: "nested"}}}, other()}, Flags: []cli.Flag{option(config{Count: 2, Label: "x"})}}
var unrelated = config{Count: 3, Label: "y"}
`,
			want: `var command = &cli.Command{
	Name: "root",
	Commands: []*cli.Command{
		{
			Name: "child",
			Examples: []string{
				"child",
			},
			Commands: []*cli.Command{
				leaf(),
				&cli.Command{
					Name: "nested",
				},
			},
		},
		other(),
	},
	Flags: []cli.Flag{
		option(config{Count: 2, Label: "x"}),
	},
}
var unrelated = config{Count: 3, Label: "y"}
`,
		},
		{
			name: "comments_and_trailing_commas",
			source: `var command = &cli.Command{ /* opening */ Name: "root", // trailing
	// examples
	Examples: []string{/* first */ "a", "b" /* last, value */}, Flags: []cli.Flag{first(), /* between */ second(),}, // flags
	Usage: "text" /* closing */}
`,
			want: `var command = &cli.Command{ /* opening */
	Name: "root", // trailing
	// examples
	Examples: []string{ /* first */
		"a",
		"b", /* last, value */
	},
	Flags: []cli.Flag{
		first(), /* between */
		second(),
	}, // flags
	Usage: "text", /* closing */
}
`,
		},
		{
			name: "empty_literals",
			source: `var command = &cli.Command{Flags: []cli.Flag{}, Commands: []*cli.Command{}, Examples: []string{}}
`,
			want: `var command = &cli.Command{
	Flags:    []cli.Flag{},
	Commands: []*cli.Command{},
	Examples: []string{},
}
`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			source := []byte(sourceHeader + tt.source)
			formatted, err := formatCommands("command.go", source)
			require.NoError(t, err)
			require.Equal(t, sourceHeader+tt.want, string(formatted))
			require.Equal(t, sourceTokens(source), sourceTokens(formatted))
			again, err := formatCommands("command.go", formatted)
			require.NoError(t, err)
			require.Equal(t, formatted, again)
		})
	}
}

func TestFormatCommandsAlreadyFormatted(t *testing.T) {
	source := []byte(sourceHeader + `var command = &cli.Command{
	Name: "root",
	Examples: []string{
		"root --help",
	},
	Flags: []cli.Flag{
		cli.String("name", "Name", cli.Required()),
	},
	Commands: []*cli.Command{
		child(),
	},
}
`)
	formatted, err := formatCommands("command.go", source)
	require.NoError(t, err)
	require.Equal(t, source, formatted)
}

func TestFormatCommandsSeparatesImportsWithoutRemovingUnusedImports(t *testing.T) {
	source := []byte(`package sample

import (
	"context"
	commands "github.com/unkeyed/unkey/pkg/cli"
)

var command = &commands.Command{Name: "root"}
`)
	formatted, err := formatCommands("command.go", source)
	require.NoError(t, err)
	require.Contains(t, string(formatted), "\"context\"\n\n\tcommands \"github.com/unkeyed/unkey/pkg/cli\"")
	require.Contains(t, string(formatted), "&commands.Command{\n\tName: \"root\",\n}")
	require.Equal(t, sourceTokens(source), sourceTokens(formatted))
}

func TestFormatCommandsIgnoresOtherTypes(t *testing.T) {
	for _, source := range []string{
		"package sample\nvar command=other.Command{Name: \"root\"}",
		sourceHeader + "var other=struct{Flags []cli.Flag}{Flags: []cli.Flag{first(), second()}}",
		"package sample\nimport cli \"example.com/cli\"\nvar command=cli.Command{Name: \"root\"}",
	} {
		formatted, err := formatCommands("command.go", []byte(source))
		require.NoError(t, err)
		require.Equal(t, source, string(formatted))
	}
}

func sourceTokens(source []byte) []string {
	var scan scanner.Scanner
	scan.Init(token.NewFileSet().AddFile("", -1, len(source)), source, nil, scanner.ScanComments)
	var tokens []string
	for {
		_, kind, value := scan.Scan()
		if kind == token.EOF {
			break
		}
		if kind == token.SEMICOLON && value == "\n" {
			continue
		}
		tokens = append(tokens, kind.String()+":"+value)
	}
	var normalized []string
	for i, value := range tokens {
		if value == ",:" {
			next := i + 1
			for next < len(tokens) && len(tokens[next]) >= 8 && tokens[next][:8] == "COMMENT:" {
				next++
			}
			if next < len(tokens) && tokens[next] == "}:" {
				continue
			}
		}
		normalized = append(normalized, value)
	}
	return normalized
}
