package util

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/unkeyed/unkey/pkg/cli"
	"golang.org/x/term"
)

// YesFlag skips only local confirmation, never API authorization or protection.
func YesFlag() *cli.BoolFlag {
	return cli.Bool("yes", "Skip confirmation; permissions and delete protection still apply")
}

// Confirm requires a terminal and explicit consent unless --yes is set.
func Confirm(cmd *cli.Command, question string) error {
	if cmd.Bool("yes") {
		return nil
	}
	question = terminalText(question)
	if !term.IsTerminal(int(os.Stdin.Fd())) || !term.IsTerminal(int(os.Stderr.Fd())) {
		return fmt.Errorf("%s\nconfirmation requires a terminal; rerun with --yes to confirm", question)
	}
	if _, err := fmt.Fprintf(os.Stderr, "%s [y/N]: ", question); err != nil {
		return err
	}
	answer, err := bufio.NewReader(io.LimitReader(os.Stdin, 16)).ReadString('\n')
	if err == io.EOF {
		return fmt.Errorf("cancelled")
	}
	if err != nil {
		return fmt.Errorf("read confirmation: %w", err)
	}
	switch strings.ToLower(strings.TrimSpace(answer)) {
	case "y", "yes":
		return nil
	default:
		return fmt.Errorf("cancelled")
	}
}
