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
	"github.com/unkeyed/unkey/pkg/cli/credentials"
	"github.com/unkeyed/unkey/pkg/cli/session"
	"github.com/unkeyed/unkey/pkg/prompt"
	"golang.org/x/term"
)

func New() *cli.Command {
	return newCommand(credentials.DefaultRootKeyStore())
}

func newCommand(keys credentials.RootKeyStore) *cli.Command {
	return &cli.Command{
		Name:        "login",
		RootAliases: []string{"login"},
		Usage:       "Sign in and store a CLI session",
		Description: `Start a device login, allow it in the browser, and store the CLI session you authorize.

The session is stored in the system keychain when that store is available, and otherwise in ~/.unkey/sessions.json. One session is kept per workspace.

Pass --root-key=- to store an existing root key without a browser. The key is read from stdin, with a hidden prompt in a terminal, so it stays out of your shell history. That flag does not read UNKEY_ROOT_KEY. Humans use unkey login. CI uses a root key.`,
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
				sleep:     nil,
				open:      openBrowser,
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
	keys      credentials.RootKeyStore
}

func run(ctx context.Context, opts options) error {
	if opts.rootKey != "" {
		return storeRootKey(opts.keys, opts.rootKey)
	}
	if opts.client == nil {
		opts.client = &http.Client{Timeout: 30 * time.Second}
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
	fmt.Printf("Open this URL, check the code, and choose permissions:\n  %s\n\n", started.VerificationURL)
	if !opts.noBrowser && opts.open != nil {
		if err := opts.open(started.VerificationURL); err != nil {
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

	apiURL := session.NormalizeAPIURL(opts.apiURL)
	if apiURL == "" {
		apiURL = session.DefaultAPIURL
	}

	if err := session.Save(session.Session{
		WorkspaceID:  issued.WorkspaceID,
		SessionID:    issued.SessionID,
		AccessToken:  issued.AccessToken,
		RefreshToken: issued.RefreshToken,
		ExpiresAt:    time.Now().Add(time.Duration(issued.ExpiresIn) * time.Second).Unix(),
		APIURL:       apiURL,
	}); err != nil {
		if revokeErr := revokeIssuedSession(ctx, opts.client, opts.apiURL, issued); revokeErr != nil {
			return fmt.Errorf("could not store the CLI session: %w", err)
		}
		return fmt.Errorf("could not store the CLI session, so it was revoked: %w", err)
	}

	fmt.Printf("Authentication successful. Session stored for workspace %s.\n", issued.WorkspaceID)
	return nil
}

func openBrowser(url string) error {
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		return nil
	}

	selected, err := prompt.New(prompt.WithWriter(os.Stderr)).SelectOrdered(
		"Open the verification URL in your browser?",
		[]prompt.SelectOption{
			{Key: "manual", Label: "Continue without opening a browser"},
			{Key: "open", Label: "Open browser"},
		},
		"manual",
	)
	if err != nil || selected != "open" {
		return err
	}

	return cli.OpenBrowser(url)
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

func storeRootKey(keys credentials.RootKeyStore, key string) error {
	if keys == nil {
		keys = credentials.DefaultRootKeyStore()
	}

	location, err := credentials.StoreRootKey(keys, key)
	if err != nil {
		return fmt.Errorf("failed to store root key: %w", err)
	}
	fmt.Printf("Authentication successful. Key stored in %s.\n", location)
	return nil
}
