package login

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/cli"
	"github.com/unkeyed/unkey/pkg/cli/keystore"
)

func TestFormatUserCode(t *testing.T) {
	require.Equal(t, "Z H V G - S X Q M", formatUserCode("zhvg-sxqm"))
	require.Equal(t, "Z H V G - S X Q M", formatUserCode("  ZHVG-SXQM  "))
}

func TestRootKeyFlagStoresTheKey(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("UNKEY_ROOT_KEY", "")
	store := &memoryRootKeyStore{}

	out := captureStdout(t, func() {
		cmd := newCommand(store)
		err := cmd.Run(context.Background(), []string{"unkey", "--root-key", "unkey_from_flag"})
		require.NoError(t, err)
	})
	require.Contains(t, out, "Authentication successful. Key stored in the system keychain.")
	require.Equal(t, "unkey_from_flag", store.secret)
	require.NotContains(t, configBody(t, home), "unkey_from_flag")
}

func TestRootKeyFlagFallsBackToTheConfigFile(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("UNKEY_ROOT_KEY", "")
	store := &memoryRootKeyStore{unavailable: true}

	path := filepath.Join(home, ".unkey", "config.toml")
	out := captureStdout(t, func() {
		cmd := newCommand(store)
		err := cmd.Run(context.Background(), []string{"unkey", "--root-key", "unkey_from_flag"})
		require.NoError(t, err)
	})
	require.Contains(t, out, "Authentication successful. Key stored in "+path)

	cfg, err := cli.LoadUserConfig(path)
	require.NoError(t, err)
	require.Equal(t, "unkey_from_flag", cfg.RootKey)
	info, err := os.Stat(path)
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0o600), info.Mode().Perm())
}

func TestDeviceLoginPollsUntilTheKeyIsIssued(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	store := &memoryRootKeyStore{}

	var mu sync.Mutex
	polls := 0
	var startedBody map[string]string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		w.Header().Set("content-type", "application/json")
		switch r.URL.Path {
		case "/v2/cli.startDeviceLogin":
			require.NoError(t, json.Unmarshal(raw, &startedBody))
			_, _ = w.Write([]byte(`{"data":{"loginId":"cdl_test","userCode":"ABCD-EFGH","verificationUriComplete":"http://dashboard.example/cli/device?user_code=ABCD-EFGH","expiresIn":300,"interval":5}}`))
		case "/v2/cli.pollDeviceLogin":
			var body map[string]string
			require.NoError(t, json.Unmarshal(raw, &body))
			require.Equal(t, "cdl_test", body["loginId"])
			mu.Lock()
			polls++
			n := polls
			mu.Unlock()
			switch n {
			case 1:
				_, _ = w.Write([]byte(`{"data":{"status":"permissions_required","interval":1}}`))
			case 2:
				_, _ = w.Write([]byte(`{"data":{"status":"permissions_required","interval":2}}`))
			case 3:
				_, _ = w.Write([]byte(`{"data":{"status":"permissions_required","interval":2}}`))
			default:
				_, _ = w.Write([]byte(`{"data":{"status":"complete","interval":2,"key":"unkey_issued","keyId":"key_1"}}`))
			}
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)

	var slept []time.Duration
	opened := ""
	err := run(context.Background(), options{
		apiURL:    srv.URL,
		noBrowser: false,
		keys:      store,
		client:    srv.Client(),
		sleep: func(_ context.Context, d time.Duration) error {
			slept = append(slept, d)
			return nil
		},
		open: func(url string) error {
			opened = url
			return nil
		},
	})
	require.NoError(t, err)
	require.NotEmpty(t, startedBody["deviceName"])
	require.Equal(t, "http://dashboard.example/cli/device?user_code=ABCD-EFGH", opened)
	require.Equal(t, []time.Duration{time.Second, 2 * time.Second, 2 * time.Second}, slept)

	require.Equal(t, "unkey_issued", store.secret)
	require.NotContains(t, configBody(t, home), "unkey_issued")
}

func TestDeviceLoginStopsWhenDenied(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("content-type", "application/json")
		if r.URL.Path == "/v2/cli.startDeviceLogin" {
			_, _ = w.Write([]byte(`{"data":{"loginId":"cdl_test","userCode":"ABCD-EFGH","verificationUriComplete":"http://dashboard.example/cli/device?user_code=ABCD-EFGH","expiresIn":300,"interval":5}}`))
			return
		}
		_, _ = w.Write([]byte(`{"data":{"status":"access_denied","interval":5,"detail":"Authorization was denied."}}`))
	}))
	t.Cleanup(srv.Close)

	err := run(context.Background(), options{
		apiURL:    srv.URL,
		noBrowser: true,
		client:    srv.Client(),
		sleep: func(context.Context, time.Duration) error {
			t.Fatal("denied login must not keep polling")
			return nil
		},
	})
	require.EqualError(t, err, "Authorization was denied.")
}

func TestDeviceLoginRetriesTransientPollErrors(t *testing.T) {
	store := &memoryRootKeyStore{}
	polls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("content-type", "application/json")
		if r.URL.Path == "/v2/cli.startDeviceLogin" {
			_, _ = w.Write([]byte(`{"data":{"loginId":"cdl_test","userCode":"ABCD-EFGH","verificationUriComplete":"http://dashboard.example/cli/device?user_code=ABCD-EFGH","expiresIn":300,"interval":1}}`))
			return
		}
		polls++
		if polls == 1 {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte(`{"error":{"detail":"try again"}}`))
			return
		}
		_, _ = w.Write([]byte(`{"data":{"status":"complete","interval":1,"key":"unkey_issued","keyId":"key_1"}}`))
	}))
	t.Cleanup(srv.Close)

	err := run(context.Background(), options{
		apiURL:    srv.URL,
		noBrowser: true,
		keys:      store,
		client:    srv.Client(),
		sleep:     func(context.Context, time.Duration) error { return nil },
	})
	require.NoError(t, err)
	require.Equal(t, 2, polls)
	require.Equal(t, "unkey_issued", store.secret)
}

func TestDeviceLoginStopsOnClientPollErrors(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("content-type", "application/json")
		if r.URL.Path == "/v2/cli.startDeviceLogin" {
			_, _ = w.Write([]byte(`{"data":{"loginId":"cdl_test","userCode":"ABCD-EFGH","verificationUriComplete":"http://dashboard.example/cli/device?user_code=ABCD-EFGH","expiresIn":300,"interval":1}}`))
			return
		}
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":{"detail":"This login was already used."}}`))
	}))
	t.Cleanup(srv.Close)

	err := run(context.Background(), options{
		apiURL:    srv.URL,
		noBrowser: true,
		client:    srv.Client(),
		sleep: func(context.Context, time.Duration) error {
			t.Fatal("client errors must not be retried")
			return nil
		},
	})
	require.EqualError(t, err, "This login was already used.")
}

func TestDeviceLoginPrintsTheKeyWhenItCannotBeStored(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("content-type", "application/json")
		if r.URL.Path == "/v2/cli.startDeviceLogin" {
			_, _ = w.Write([]byte(`{"data":{"loginId":"cdl_test","userCode":"ABCD-EFGH","verificationUriComplete":"http://dashboard.example/cli/device?user_code=ABCD-EFGH","expiresIn":300,"interval":1}}`))
			return
		}
		_, _ = w.Write([]byte(`{"data":{"status":"complete","interval":1,"key":"unkey_issued","keyId":"key_1"}}`))
	}))
	t.Cleanup(srv.Close)

	stderr := captureStderr(t, func() {
		err := run(context.Background(), options{
			apiURL:    srv.URL,
			noBrowser: true,
			keys:      &memoryRootKeyStore{setErr: errors.New("keychain locked")},
			client:    srv.Client(),
			sleep:     func(context.Context, time.Duration) error { return nil },
		})
		require.ErrorContains(t, err, "keychain locked")
	})
	require.Contains(t, stderr, "key_1")
	require.Contains(t, stderr, "unkey_issued")
}

func TestReadRootKeyFromStdin(t *testing.T) {
	key, err := readRootKeyFrom(strings.NewReader("  unkey_piped\nignored\n"))
	require.NoError(t, err)
	require.Equal(t, "unkey_piped", key)

	_, err = readRootKeyFrom(strings.NewReader("\n"))
	require.EqualError(t, err, "no root key on stdin")
}

type memoryRootKeyStore struct {
	secret      string
	unavailable bool
	setErr      error
}

func (m *memoryRootKeyStore) Get() (string, error) {
	if m.unavailable {
		return "", keystore.ErrUnavailable
	}
	if m.secret == "" {
		return "", keystore.ErrNotFound
	}
	return m.secret, nil
}

func (m *memoryRootKeyStore) Set(secret string) error {
	if m.setErr != nil {
		return m.setErr
	}
	if m.unavailable {
		return keystore.ErrUnavailable
	}
	m.secret = secret
	return nil
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	require.NoError(t, err)
	original := os.Stdout
	os.Stdout = w
	t.Cleanup(func() { os.Stdout = original })
	fn()
	os.Stdout = original
	require.NoError(t, w.Close())
	out, err := io.ReadAll(r)
	require.NoError(t, err)
	return string(out)
}

func captureStderr(t *testing.T, fn func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	require.NoError(t, err)
	original := os.Stderr
	os.Stderr = w
	t.Cleanup(func() { os.Stderr = original })
	fn()
	os.Stderr = original
	require.NoError(t, w.Close())
	out, err := io.ReadAll(r)
	require.NoError(t, err)
	return string(out)
}

func configBody(t *testing.T, home string) string {
	t.Helper()
	body, err := os.ReadFile(filepath.Join(home, ".unkey", "config.toml"))
	if os.IsNotExist(err) {
		return ""
	}
	require.NoError(t, err)
	return string(body)
}
