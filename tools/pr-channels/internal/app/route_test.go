package app

import (
	"strings"
	"testing"
)

func TestChannelName(t *testing.T) {
	cases := []struct {
		repo   string
		number int
		branch string
		want   string
	}{
		{"unkey", 1234, "main", "pr-unkey-1234-main"},
		{"Unkey", 1234, "Feature/Foo", "pr-unkey-1234-feature-foo"},
		{"my.repo", 7, "eng-2892-billing", "pr-my-repo-7-eng-2892-billing"},
		{"with spaces", 9, "", "pr-with-spaces-9"},
	}
	for _, c := range cases {
		if got := ChannelName(c.repo, c.number, c.branch); got != c.want {
			t.Errorf("ChannelName(%q,%d,%q)=%q want %q", c.repo, c.number, c.branch, got, c.want)
		}
	}
}

func TestChannelNameTruncatesbutKeepsNumber(t *testing.T) {
	// Long repo: even pr-<repo>-<number> overflows, so the branch is dropped and
	// the repo trimmed, but the number survives.
	got := ChannelName(strings.Repeat("a", 200), 42, "some-branch")
	if len(got) > maxChannelName {
		t.Fatalf("name too long: %d", len(got))
	}
	if !strings.HasSuffix(got, "-42") {
		t.Fatalf("lost PR number: %q", got)
	}

	// Long branch: the branch is truncated to fit while pr-<repo>-<number> stays.
	got = ChannelName("unkey", 7, strings.Repeat("b", 200))
	if len(got) > maxChannelName {
		t.Fatalf("name too long: %d", len(got))
	}
	if !strings.HasPrefix(got, "pr-unkey-7-") {
		t.Fatalf("lost repo/number prefix: %q", got)
	}
}

func TestIsBotAuthor(t *testing.T) {
	bots := []struct {
		login, typ string
	}{
		{"dependabot[bot]", "Bot"},
		{"renovate", "User"},
		{"dependabot", "User"},
		{"some-app[bot]", "User"},
	}
	for _, b := range bots {
		if !IsBotAuthor(b.login, b.typ) {
			t.Errorf("expected %q/%q to be a bot", b.login, b.typ)
		}
	}
	if IsBotAuthor("chronark", "User") {
		t.Error("human flagged as bot")
	}
}

func TestShouldOpenChannel(t *testing.T) {
	cases := []struct {
		action       string
		draft        bool
		isBot        bool
		hasReviewers bool
		want         bool
	}{
		{"opened", false, false, true, true},   // ready + reviewers
		{"opened", false, false, false, false}, // ready but no reviewer -> no channel
		{"opened", true, false, true, false},   // draft
		{"ready_for_review", true, false, true, true},
		{"ready_for_review", false, false, false, false},
		{"review_requested", false, false, false, true}, // reviewer just asked
		{"review_requested", true, false, false, true},  // even on a draft
		{"opened", false, true, true, false},            // bot
		{"reopened", false, false, true, true},
		{"synchronize", false, false, true, false},
		{"closed", false, false, true, false},
	}
	for _, c := range cases {
		if got := ShouldOpenChannel(c.action, c.draft, c.isBot, c.hasReviewers); got != c.want {
			t.Errorf("ShouldOpenChannel(%q,draft=%v,bot=%v,reviewers=%v)=%v want %v",
				c.action, c.draft, c.isBot, c.hasReviewers, got, c.want)
		}
	}
}

func TestShouldPostOverview(t *testing.T) {
	cases := []struct {
		action string
		draft  bool
		isBot  bool
		want   bool
	}{
		{"opened", false, false, true},
		{"opened", true, false, false}, // draft not in feed yet
		{"ready_for_review", true, false, true},
		{"opened", false, true, false}, // bot excluded
		{"reopened", false, false, true},
		{"synchronize", false, false, false},
		{"closed", false, false, false}, // closes ignored; merges handled elsewhere
	}
	for _, c := range cases {
		if got := ShouldPostOverview(c.action, c.draft, c.isBot); got != c.want {
			t.Errorf("ShouldPostOverview(%q,draft=%v,bot=%v)=%v want %v",
				c.action, c.draft, c.isBot, got, c.want)
		}
	}
}

func TestMention(t *testing.T) {
	if got := Mention("chronark", "U123"); got != "<@U123>" {
		t.Errorf("mapped mention = %q", got)
	}
	if got := Mention("chronark", ""); got != "`@chronark`" {
		t.Errorf("unmapped mention = %q", got)
	}
}
