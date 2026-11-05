package keyleak

import (
	"testing"

	"github.com/stretchr/testify/require"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/email"
	"github.com/unkeyed/unkey/pkg/uid"
)

// TestLeakEmailMatchesReporterAndKeyKind guarantees that GitHub leaks keep
// their original template and variables for every key kind, including
// notifications enqueued before the kind existed, and that a TruffleHog leak
// tells root key owners and API key issuers different things. A TruffleHog
// leak without a kind fails instead of sending root key advice for a key the
// workspace issued to its own user.
func TestLeakEmailMatchesReporterAndKeyKind(t *testing.T) {
	github := hydrav1.KeyLeakReporter_KEY_LEAK_REPORTER_GITHUB
	trufflehog := hydrav1.KeyLeakReporter_KEY_LEAK_REPORTER_TRUFFLEHOG
	unspecified := hydrav1.KeyLeakKeyKind_KEY_LEAK_KEY_KIND_UNSPECIFIED
	root := hydrav1.KeyLeakKeyKind_KEY_LEAK_KEY_KIND_ROOT
	api := hydrav1.KeyLeakKeyKind_KEY_LEAK_KEY_KIND_API

	for _, test := range []struct {
		name     string
		reporter hydrav1.KeyLeakReporter
		kind     hydrav1.KeyLeakKeyKind
		template string
		keyID    bool
	}{
		{name: "legacy GitHub", reporter: hydrav1.KeyLeakReporter_KEY_LEAK_REPORTER_UNSPECIFIED, kind: unspecified, template: "secret-scanning-key-detected"},
		{name: "GitHub without kind", reporter: github, kind: unspecified, template: "secret-scanning-key-detected"},
		{name: "GitHub root key", reporter: github, kind: root, template: "secret-scanning-key-detected"},
		{name: "GitHub API key", reporter: github, kind: api, template: "secret-scanning-key-detected"},
		{name: "TruffleHog root key", reporter: trufflehog, kind: root, template: "trufflehog-key-detected"},
		{name: "TruffleHog API key", reporter: trufflehog, kind: api, template: "trufflehog-api-key-detected", keyID: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			leak := &hydrav1.KeyLeakNotification{
				KeyId: uid.New(uid.KeyPrefix), OrgId: uid.New(uid.OrgPrefix), Reporter: test.reporter, KeyKind: test.kind,
				Source: "Docker", Url: "https://hub.docker.com/r/unkeyed/example", Date: "Thu Nov 05 2026",
			}
			idempotencyKey := "leaked-key:" + uid.New(uid.TestPrefix)
			message, err := leakEmail(&hydrav1.KeyLeakEmailRequest{Leak: leak, Email: "first@example.com"}, idempotencyKey)
			require.NoError(t, err)

			variables := map[string]string{"DATE": leak.Date, "SOURCE": leak.Source, "URL": leak.Url}
			if test.keyID {
				variables["KEY_ID"] = leak.KeyId
			}
			require.Equal(t, email.Email{
				To: []string{"first@example.com"}, TemplateID: test.template, Variables: variables, IdempotencyKey: idempotencyKey,
			}, message)
		})
	}

	t.Run("TruffleHog without kind", func(t *testing.T) {
		_, err := leakEmail(&hydrav1.KeyLeakEmailRequest{
			Leak:  &hydrav1.KeyLeakNotification{KeyId: uid.New(uid.KeyPrefix), Reporter: trufflehog, KeyKind: unspecified},
			Email: "first@example.com",
		}, "leaked-key:"+uid.New(uid.TestPrefix))
		require.Error(t, err)
	})
}
