package util

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/unkeyed/unkey/pkg/cli"
	"golang.org/x/term"
)

// YesFlag skips only local confirmation, never API authorization or protection.
func YesFlag() *cli.BoolFlag {
	return cli.Bool("yes", "Skip confirmation for complete commands; permissions and delete protection still apply")
}

// Confirm requires terminal consent unless an unguided caller supplies --yes.
func Confirm(ctx context.Context, cmd *cli.Command, question string) error {
	confirm, guided := ctx.Value(confirmationKey{}).(func(string) error)
	if cmd.Bool("yes") && (!guided || confirm == nil) {
		return nil
	}
	question = terminalText(question)
	if cmd.Bool("no-input") {
		return fmt.Errorf("%s\nconfirmation is disabled by --no-input; use --yes to confirm", question)
	}
	if !term.IsTerminal(int(os.Stdin.Fd())) || !term.IsTerminal(int(os.Stderr.Fd())) {
		return fmt.Errorf("%s\nconfirmation requires a terminal; rerun with --yes to confirm", question)
	}
	if guided && confirm != nil {
		return confirm(question)
	}
	if _, err := fmt.Fprintf(os.Stderr, "%s [y/N]: ", question); err != nil {
		return err
	}
	answer, err := bufio.NewReader(io.LimitReader(os.Stdin, 16)).ReadString('\n')
	if err == io.EOF {
		return &CancellationError{Message: "cancelled; no request was sent"}
	}
	if err != nil {
		return fmt.Errorf("read confirmation: %w", err)
	}
	switch strings.ToLower(strings.TrimSpace(answer)) {
	case "y", "yes":
		return nil
	default:
		return &CancellationError{Message: "cancelled; no request was sent"}
	}
}

// WithConfirmation overrides the prompt without bypassing Confirm's safety checks.
func WithConfirmation(ctx context.Context, confirm func(string) error) context.Context {
	return context.WithValue(ctx, confirmationKey{}, confirm)
}

type confirmationKey struct{}
