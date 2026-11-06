package keyleak

import hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"

type reporter struct {
	name    string
	finding string
	keyNoun func(leak *hydrav1.KeyLeakNotification) string
	emails  map[hydrav1.KeyLeakKeyKind]emailTemplate
}

type emailTemplate struct {
	id    string
	keyID bool
}

var gitHubEmail = emailTemplate{id: "secret-scanning-key-detected", keyID: false}

// Notifications GitHub enqueued before the key kind existed carry UNSPECIFIED.
var (
	reporterGitHub = reporter{
		name: "GitHub", finding: "exposed on GitHub", keyNoun: gitHubKeyNoun,
		emails: map[hydrav1.KeyLeakKeyKind]emailTemplate{
			hydrav1.KeyLeakKeyKind_KEY_LEAK_KEY_KIND_UNSPECIFIED: gitHubEmail,
			hydrav1.KeyLeakKeyKind_KEY_LEAK_KEY_KIND_ROOT:        gitHubEmail,
			hydrav1.KeyLeakKeyKind_KEY_LEAK_KEY_KIND_API:         gitHubEmail,
		},
	}
	reporterTruffleHog = reporter{
		name: "TruffleHog", finding: "reported by TruffleHog", keyNoun: keyKindNoun,
		emails: map[hydrav1.KeyLeakKeyKind]emailTemplate{
			hydrav1.KeyLeakKeyKind_KEY_LEAK_KEY_KIND_ROOT: {id: "trufflehog-key-detected", keyID: false},
			hydrav1.KeyLeakKeyKind_KEY_LEAK_KEY_KIND_API:  {id: "trufflehog-api-key-detected", keyID: true},
		},
	}
)

// Notifications enqueued before the reporter field existed carry UNSPECIFIED
// and all came from GitHub.
func reporterOf(leak *hydrav1.KeyLeakNotification) reporter {
	switch leak.GetReporter() {
	case hydrav1.KeyLeakReporter_KEY_LEAK_REPORTER_TRUFFLEHOG:
		return reporterTruffleHog
	case hydrav1.KeyLeakReporter_KEY_LEAK_REPORTER_UNSPECIFIED, hydrav1.KeyLeakReporter_KEY_LEAK_REPORTER_GITHUB:
		return reporterGitHub
	}
	return reporterGitHub
}

func gitHubKeyNoun(leak *hydrav1.KeyLeakNotification) string {
	switch leak.GetType() {
	case "unkey_root_key":
		return "Root key"
	case "unkey_key":
		return "API key"
	}
	return "Key"
}

func keyKindNoun(leak *hydrav1.KeyLeakNotification) string {
	switch leak.GetKeyKind() {
	case hydrav1.KeyLeakKeyKind_KEY_LEAK_KEY_KIND_ROOT:
		return "Root key"
	case hydrav1.KeyLeakKeyKind_KEY_LEAK_KEY_KIND_API:
		return "API key"
	case hydrav1.KeyLeakKeyKind_KEY_LEAK_KEY_KIND_UNSPECIFIED:
	}
	return "Key"
}
