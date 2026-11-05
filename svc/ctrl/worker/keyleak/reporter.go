package keyleak

import hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"

type reporter struct {
	name    string
	finding string
}

var (
	reporterGitHub     = reporter{name: "GitHub", finding: "exposed on GitHub"}
	reporterTruffleHog = reporter{name: "TruffleHog", finding: "reported by TruffleHog"}
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
