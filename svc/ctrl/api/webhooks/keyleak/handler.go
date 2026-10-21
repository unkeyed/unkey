package keyleak

import (
	"context"
	"fmt"
	"net/http"
	"time"

	restate "github.com/restatedev/sdk-go"
	"github.com/restatedev/sdk-go/ingress"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/hash"
	"github.com/unkeyed/unkey/pkg/webhook"
	"github.com/unkeyed/unkey/pkg/webhook/verifiers/githubsecrets"
)

const bodyBytesMax = 4 << 20

type handler struct {
	restate *ingress.Client
}

func New(client *ingress.Client, verifier webhook.Verifier) http.Handler {
	h := &handler{restate: client}

	receiver := webhook.New("github_secret_scanning", verifier, webhook.WithMaxBodySize(bodyBytesMax)).
		On([]string{githubsecrets.EventType}, webhook.Typed(h.report))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		w.Header().Set("Cache-Control", "no-store")
		receiver.ServeHTTP(w, r.WithContext(ctx))
	})
}

type reportedToken struct {
	Token  string `json:"token"`
	Type   string `json:"type"`
	URL    string `json:"url"`
	Source string `json:"source"`
}

func (h *handler) report(ctx context.Context, event webhook.Event, tokens []reportedToken) error {
	if tokens == nil {
		return webhook.ErrBadRequest
	}
	for _, token := range tokens {
		if token.Token == "" || token.Type == "" || token.Source == "" {
			return webhook.ErrBadRequest
		}
	}

	if len(tokens) == 0 {
		return nil
	}

	report := &hydrav1.KeyLeakReport{Tokens: make([]*hydrav1.KeyLeakToken, 0, len(tokens))}
	for _, token := range tokens {
		report.Tokens = append(report.Tokens, &hydrav1.KeyLeakToken{
			Hash:   hash.Sha256(token.Token),
			Type:   token.Type,
			Url:    token.URL,
			Source: token.Source,
		})
	}

	_, err := hydrav1.NewKeyLeakServiceIngressClient(h.restate).Process().Send(ctx, report,
		restate.WithIdempotencyKey(hash.Sha256(string(event.Payload))))
	if err != nil {
		return fmt.Errorf("%w: persist key leak report: %w", webhook.ErrUnavailable, err)
	}

	return nil
}
