package cli

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/uid"
)

func TestRunHandlesEachFailureAtTheInnermostBoundary(t *testing.T) {
	problem := errors.New("operation failed")
	for _, test := range []struct {
		name        string
		args        []string
		prepareErr  error
		actionErr   error
		want        string
		cause       error
		kind        InputErrorKind
		flags       []string
		prepareRuns int
		actionRuns  int
		unparsed    bool
	}{
		{name: "unknown", args: []string{"--unknown", "--count=1"}, want: "unknown flag: unknown", kind: InputErrorUnknownFlag, flags: []string{"unknown"}, unparsed: true},
		{name: "missing value", args: []string{"--count"}, want: "flag count requires a value", kind: InputErrorMissingFlagValue, flags: []string{"count"}, unparsed: true},
		{name: "invalid type", args: []string{"--count=invalid"}, want: "expected an integer", cause: ErrInvalidIntValue, kind: InputErrorInvalidFlagValue, flags: []string{"count"}, unparsed: true},
		{name: "required", want: "required flag missing: target", kind: InputErrorMissingRequiredFlag, flags: []string{"target"}, prepareRuns: 1},
		{name: "conflict", args: []string{"--target=value", "--body=value"}, want: "mutually exclusive", kind: InputErrorConflictingFlags, flags: []string{"target", "body"}},
		{name: "prepare", args: []string{"--target=value"}, prepareErr: problem, want: problem.Error(), cause: problem, prepareRuns: 1},
		{name: "action", args: []string{"--target=value"}, actionErr: problem, want: problem.Error(), cause: problem, prepareRuns: 1, actionRuns: 1},
		{name: "success", args: []string{"--target=value"}, prepareRuns: 1, actionRuns: 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			prepareRuns, actionRuns, handled := 0, 0, 0
			var received error
			var receivedArgs []string
			command := &Command{
				Name: "execute",
				Flags: []Flag{
					String("target", "", Required(), MutuallyExclusive("body")),
					String("body", ""), Int("count", ""),
				},
				Prepare: func(context.Context, *Command) error {
					prepareRuns++
					return test.prepareErr
				},
				Action: func(context.Context, *Command) error {
					actionRuns++
					return test.actionErr
				},
				OnError: func(_ context.Context, cmd *Command, args []string, err error) error {
					handled++
					received, receivedArgs = err, args
					require.Equal(t, "execute", cmd.Name)
					return fmt.Errorf("rendered: %w", err)
				},
			}
			app := &Command{
				Name: "tool",
				OnError: func(_ context.Context, _ *Command, _ []string, err error) error {
					t.Error("ancestor hook must not format a leaf failure again")
					return err
				},
				Commands: []*Command{{Name: "group", Commands: []*Command{command}}},
			}
			err := app.Run(t.Context(), append([]string{"tool", "group", "execute"}, test.args...))
			if test.want == "" {
				require.NoError(t, err)
				require.Zero(t, handled)
			} else {
				require.ErrorContains(t, err, "rendered: ")
				require.ErrorContains(t, received, test.want)
				require.Equal(t, 1, handled)
				if test.cause != nil {
					require.ErrorIs(t, received, test.cause)
				}
				if test.kind != "" {
					var input *InputError
					require.ErrorAs(t, received, &input)
					require.Equal(t, test.kind, input.Kind)
					require.Equal(t, test.flags, input.Flags)
				}
				if test.unparsed {
					require.Equal(t, test.args, receivedArgs)
				} else {
					require.Nil(t, receivedArgs)
				}
			}
			require.Equal(t, test.prepareRuns, prepareRuns)
			require.Equal(t, test.actionRuns, actionRuns)
		})
	}
}

func TestRunInheritsNearestErrorHook(t *testing.T) {
	problem := errors.New("action failed")
	for _, path := range [][]string{{"group", "execute"}, {"alias"}, {"group", "unknown"}} {
		t.Run(fmt.Sprint(path), func(t *testing.T) {
			calls := 0
			group := &Command{
				Name: "group",
				OnError: func(_ context.Context, _ *Command, _ []string, err error) error {
					calls++
					return err
				},
				Commands: []*Command{{
					Name: "execute", RootAliases: []string{"alias"},
					Action: func(context.Context, *Command) error { return problem },
				}},
			}
			root := &Command{Name: "tool", Commands: []*Command{group}}
			err := root.Run(t.Context(), append([]string{"tool"}, path...))
			require.Error(t, err)
			require.Equal(t, 1, calls)
		})
	}
}

func TestRunNeverEchoesSuppliedValuesInParseDiagnostics(t *testing.T) {
	secret := uid.New(uid.KeyPrefix)
	for _, flag := range []Flag{
		Bool("value", ""), Int("value", ""), Int64("value", ""), Float("value", ""),
		Duration("value", ""), Enum("value", "", []string{"human", "json"}),
		String("value", "", Validate(func(value string) error { return fmt.Errorf("rejected: %s", value) })),
	} {
		t.Run(fmt.Sprintf("%T", flag), func(t *testing.T) {
			cmd := &Command{Name: "tool", Flags: []Flag{flag}}
			err := cmd.Run(t.Context(), []string{"tool", "--value=" + secret})
			require.ErrorContains(t, err, "--value")
			require.NotContains(t, err.Error(), secret)
		})
	}
	for _, args := range [][]string{{"--unknown=" + secret}, {"--", secret}} {
		cmd := &Command{Name: "tool"}
		err := cmd.Run(t.Context(), append([]string{"tool"}, args...))
		require.Error(t, err)
		require.NotContains(t, err.Error(), secret)
	}
}
