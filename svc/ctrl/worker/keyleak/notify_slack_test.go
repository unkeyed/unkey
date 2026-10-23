package keyleak_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/testutil/containers"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/ctrl/internal/slack"
	"github.com/unkeyed/unkey/svc/ctrl/worker/keyleak"
)

func TestNotifySlackFormatsLeakAndWorkspaceMembers(t *testing.T) {
	messages := make(chan slack.Payload, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var message slack.Payload
		if err := json.NewDecoder(r.Body).Decode(&message); err != nil {
			t.Error(err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		messages <- message
	}))
	t.Cleanup(server.Close)

	tests := []struct {
		name    string
		keyType string
		url     string
		members []string
		err     error
		want    []string
		absent  []string
	}{
		{
			name: "root key with members", keyType: "unkey_root_key",
			url:     "https://github.com/unkeyed/example/blob/abc/config.env?view=1&line=2#L4",
			members: []string{"first@example.com", "second@example.com"},
			want: []string{"Root key exposed on GitHub", "*Workspace members*", "<mailto:first@example.com|first@example.com>", "second@example.com",
				"<https://github.com/unkeyed/example/blob/abc/config.env?view=1&amp;line=2#L4|View exposed file>"},
		},
		{
			name: "API key without members", keyType: "unkey_key",
			url:    "https://github.com/unkeyed/example/commit/abc",
			want:   []string{"API key exposed on GitHub", "No member email addresses found", "|View source>"},
			absent: []string{"Root key exposed", "Workspace members unavailable"},
		},
		{
			name: "member lookup fails", keyType: "unkey_root_key",
			err:  errors.New("member provider unavailable"),
			want: []string{"Workspace members unavailable", "Source URL unavailable"},
		},
		{
			name: "many members", keyType: "unkey_root_key",
			members: []string{"first@example.com", "second@example.com", "third@example.com", "fourth@example.com", "fifth@example.com"},
			want:    []string{"third@example.com", "and 2 more members"},
			absent:  []string{"fourth@example.com", "fifth@example.com"},
		},
		{
			name: "link delimiters cannot inject mentions", keyType: "unkey_root_key",
			url:     "https://github.com/unkeyed/example/blob/abc/file|label><!channel>",
			members: []string{"member&team@example.com"},
			want:    []string{"<mailto:member&amp;team@example.com|member&amp;team@example.com>", "%7Clabel%3E%3C%21channel%3E|View exposed file>"},
			absent:  []string{"<!channel>", "|label>"},
		},
		{
			name: "unsupported URL", keyType: "unknown",
			url:    "javascript:alert(1)",
			want:   []string{"Key exposed on GitHub", "Source URL unavailable"},
			absent: []string{"javascript:", "Root key exposed"},
		},
		{
			name: "oversized URL", keyType: "unkey_root_key",
			url:  "https://github.com/unkeyed/example/blob/abc/" + strings.Repeat("a", 3000),
			want: []string{"Source URL unavailable"},
		},
		{
			name: "long source keeps link within Slack limits", keyType: "unkey_root_key",
			url:  "https://github.com/unkeyed/example/blob/abc/" + strings.Repeat("&", 240) + "?query=" + strings.Repeat("a", 700),
			want: []string{"|View exposed file>"},
		},
		{
			name: "oversized member data stays within Slack limits", keyType: "unkey_root_key",
			members: []string{strings.Repeat("&", 1000), "<!here>", "last@example.com"},
			want:    []string{"&lt;!here&gt;", "last@example.com"},
			absent:  []string{"mailto:&", "<!here>"},
		},
	}
	organizations := make(map[string]int, len(tests))
	orgIDs := make([]string, len(tests))
	for i := range tests {
		orgIDs[i] = uid.New(uid.TestPrefix)
		organizations[orgIDs[i]] = i
	}
	service := keyleak.New(keyleak.Config{
		Members: memberLookup(func(_ context.Context, orgID string) ([]string, error) {
			i, ok := organizations[orgID]
			if !ok {
				return nil, errors.New("unexpected organization")
			}
			test := tests[i]
			return test.members, test.err
		}),
		SlackWebhookURL: server.URL,
	})
	runtime := containers.Restate(t, service.Definition())
	client := hydrav1.NewKeyLeakServiceIngressClient(runtime.IngressClient)

	for i, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			leak := &hydrav1.KeyLeakNotification{
				KeyId: uid.New(uid.KeyPrefix), OrgId: orgIDs[i], WorkspaceName: "R&D <!here> @channel",
				Type: test.keyType, Source: "content", Url: test.url, Date: "Thu Oct 22 2026",
			}
			if strings.Contains(test.url, "/commit/") {
				leak.Source = "commit"
			}
			ctx, cancel := context.WithTimeout(t.Context(), 20*time.Second)
			t.Cleanup(cancel)
			_, err := client.NotifySlack().Request(ctx, leak)
			require.NoError(t, err)
			select {
			case message := <-messages:
				require.NotEmpty(t, message.Blocks)
				require.Equal(t, "header", message.Blocks[0].Type)
				var text strings.Builder
				text.WriteString(message.Text)
				for _, block := range message.Blocks {
					if block.Text != nil {
						text.WriteString(block.Text.Text)
						require.LessOrEqual(t, len([]rune(block.Text.Text)), 3000)
						if block.Text.Type == "mrkdwn" {
							require.True(t, block.Text.Verbatim, "untrusted text must not trigger automatic mentions")
						}
					}
					for _, field := range block.Fields {
						text.WriteString(field.Text)
						require.LessOrEqual(t, len([]rune(field.Text)), 2000)
						require.True(t, field.Verbatim)
					}
					for _, element := range block.Elements {
						text.WriteString(element.Text)
						require.LessOrEqual(t, len([]rune(element.Text)), 3000)
					}
				}
				require.Contains(t, text.String(), "R&amp;D &lt;!here&gt; @channel")
				require.NotContains(t, text.String(), "<!here>")
				require.Contains(t, text.String(), leak.KeyId)
				require.Contains(t, text.String(), leak.OrgId)
				require.Contains(t, text.String(), leak.Date)
				require.Contains(t, text.String(), "Rotate or revoke this key.")
				for _, want := range test.want {
					require.Contains(t, text.String(), want)
				}
				for _, absent := range test.absent {
					require.NotContains(t, text.String(), absent)
				}
			case <-ctx.Done():
				t.Fatal("Slack notification was not delivered")
			}
		})
	}
}

type memberLookup func(context.Context, string) ([]string, error)

func (f memberLookup) MemberEmails(ctx context.Context, orgID string) ([]string, error) {
	return f(ctx, orgID)
}
