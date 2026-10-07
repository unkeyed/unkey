package login

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/unkeyed/unkey/cmd/api/util"
	"github.com/unkeyed/unkey/pkg/cli"
	"golang.org/x/term"
)

func New() *cli.Command {
	return newCommand(cli.DefaultRootKeyStore())
}

func newCommand(keys cli.RootKeyStore) *cli.Command {
	return &cli.Command{
		Name:  "login",
		Usage: "Sign in and store a root key",
		Description: `Start a device login, allow it in the browser, and store the root key you authorize.

The key is stored in the system keychain when that store is available, and otherwise in ~/.unkey/config.toml. It is not printed.

Pass --root-key=- to store an existing key without a browser. The key is read from stdin, with a hidden prompt in a terminal, so it stays out of your shell history. That flag does not read UNKEY_ROOT_KEY.`,
		Examples: []string{
			"unkey login",
			"unkey login --no-browser",
			"unkey login --root-key=-",
			"printf '%s' \"$UNKEY_ROOT_KEY\" | unkey login --root-key=-",
		},
		Flags: []cli.Flag{
			util.APIURLFlag(),
			cli.String("root-key", "Store a root key and skip browser login. Use - to read it from stdin"),
			cli.Bool("no-browser", "Print the verification URL without opening a browser"),
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			rootKey := strings.TrimSpace(cmd.String("root-key"))
			if rootKey == "-" {
				read, err := readRootKey(os.Stdin)
				if err != nil {
					return err
				}
				rootKey = read
			}
			return run(ctx, options{
				apiURL:    cmd.String("api-url"),
				rootKey:   rootKey,
				noBrowser: cmd.Bool("no-browser"),
				client:    &http.Client{Timeout: 30 * time.Second},
				sleep:     sleepContext,
				open:      cli.OpenBrowser,
				keys:      keys,
			})
		},
	}
}

type options struct {
	apiURL    string
	rootKey   string
	noBrowser bool
	client    *http.Client
	sleep     func(context.Context, time.Duration) error
	open      func(string) error
	keys      cli.RootKeyStore
}

func run(ctx context.Context, opts options) error {
	if opts.rootKey != "" {
		return storeRootKey(opts.keys, opts.rootKey)
	}
	if opts.client == nil {
		opts.client = &http.Client{Timeout: 30 * time.Second}
	}
	if opts.sleep == nil {
		opts.sleep = sleepContext
	}
	deviceName, err := os.Hostname()
	if err != nil {
		deviceName = ""
	}
	started, err := startLogin(ctx, opts.client, opts.apiURL, deviceName)
	if err != nil {
		return err
	}
	fmt.Printf("Your confirmation code:\n\n  %s\n\n", formatUserCode(started.UserCode))
	fmt.Printf("Open this URL, check the code, and choose permissions:\n  %s\n\n", started.VerificationURIComplete)
	if !opts.noBrowser && opts.open != nil {
		if err := opts.open(started.VerificationURIComplete); err != nil {
			fmt.Printf("Could not open a browser (%s). Use the URL above.\n\n", err.Error())
		}
	}
	deadline := time.Time{}
	if started.ExpiresIn > 0 {
		deadline = time.Now().Add(time.Duration(started.ExpiresIn) * time.Second)
	}
	issued, err := pollUntilKey(ctx, opts.client, opts.apiURL, started.LoginID, deadline, time.Duration(started.Interval)*time.Second, opts.sleep)
	if err != nil {
		return err
	}
	if err := storeRootKey(opts.keys, issued.Key); err != nil {
		// The server never returns this key again, so losing it here would leave
		// a live root key that nobody holds.
		fmt.Fprintf(os.Stderr, "%s\n\nSave this root key (%s) now. It will not be shown again:\n\n  %s\n\n", err.Error(), issued.KeyID, issued.Key)
		return err
	}
	return nil
}

func readRootKey(stdin *os.File) (string, error) {
	if term.IsTerminal(int(stdin.Fd())) {
		fmt.Fprint(os.Stderr, "Root key: ")
		raw, err := term.ReadPassword(int(stdin.Fd()))
		fmt.Fprintln(os.Stderr)
		if err != nil {
			return "", fmt.Errorf("read root key: %w", err)
		}
		return strings.TrimSpace(string(raw)), nil
	}
	return readRootKeyFrom(stdin)
}

func readRootKeyFrom(r io.Reader) (string, error) {
	line, err := bufio.NewReader(io.LimitReader(r, 4096)).ReadString('\n')
	if err != nil && err != io.EOF {
		return "", fmt.Errorf("read root key: %w", err)
	}
	key := strings.TrimSpace(line)
	if key == "" {
		return "", fmt.Errorf("no root key on stdin")
	}
	return key, nil
}

func storeRootKey(keys cli.RootKeyStore, key string) error {
	if keys == nil {
		keys = cli.DefaultRootKeyStore()
	}
	location, err := cli.StoreRootKey(keys, key)
	if err != nil {
		return fmt.Errorf("failed to store root key: %w", err)
	}
	fmt.Printf("Authentication successful. Key stored in %s.\n", location)
	return nil
}

func sleepContext(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
