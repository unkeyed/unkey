package login

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/unkeyed/unkey/cmd/api/util"
	"github.com/unkeyed/unkey/pkg/cli"
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

Pass --root-key to store an existing key without a browser. That flag does not read UNKEY_ROOT_KEY.`,
		Examples: []string{
			"unkey login",
			"unkey login --no-browser",
			"unkey login --root-key=unkey_xxx",
		},
		Flags: []cli.Flag{
			util.APIURLFlag(),
			cli.String("root-key", "Store this root key and skip browser login"),
			cli.Bool("no-browser", "Print the verification URL without opening a browser"),
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return run(ctx, options{
				apiURL:    cmd.String("api-url"),
				rootKey:   strings.TrimSpace(cmd.String("root-key")),
				noBrowser: cmd.Bool("no-browser"),
				client:    &http.Client{Timeout: 30 * time.Second},
				sleep:     sleepContext,
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
	key, err := pollUntilKey(ctx, opts.client, opts.apiURL, started.LoginID, deadline, time.Duration(started.Interval)*time.Second, opts.sleep)
	if err != nil {
		return err
	}
	return storeRootKey(opts.keys, key)
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
