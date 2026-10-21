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
		OnResponse([]string{githubsecrets.EventType}, webhook.TypedResponse(http.StatusCreated, h.report))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 25*time.Second)
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

type result struct {
	TokenRaw  string `json:"token_raw,omitempty"`
	TokenHash string `json:"token_hash,omitempty"`
	TokenType string `json:"token_type"`
	Label     string `json:"label"`
}

func (h *handler) report(ctx context.Context, event webhook.Event, tokens []reportedToken) ([]result, error) {
	if tokens == nil {
		return nil, webhook.ErrBadRequest
	}
	for _, token := range tokens {
		if token.Token == "" || token.Type == "" || token.Source == "" {
			return nil, webhook.ErrBadRequest
		}
	}

	results := make([]result, 0, len(tokens))
	if len(tokens) == 0 {
		return results, nil
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

	invocation, err := hydrav1.NewKeyLeakServiceIngressClient(h.restate).Process().Send(ctx, report,
		restate.WithIdempotencyKey(hash.Sha256(string(event.Payload))))
	if err != nil {
		return nil, fmt.Errorf("%w: persist key leak report: %w", webhook.ErrUnavailable, err)
	}
	classified, err := invocation.Attach(ctx)
	if err != nil {
		return nil, fmt.Errorf("%w: classify key leak report: %w", webhook.ErrUnavailable, err)
	}

	matched := make(map[string]bool, len(classified.GetMatchedHashes()))
	for _, hashed := range classified.GetMatchedHashes() {
		matched[hashed] = true
	}
	for i, token := range tokens {
		hashed := report.GetTokens()[i].GetHash()
		if !matched[hashed] {
			results = append(results, result{
				TokenRaw:  "",
				TokenHash: hashed,
				TokenType: token.Type,
				Label:     "false_positive",
			})
			continue
		}

		results = append(results, result{
			TokenRaw:  token.Token,
			TokenHash: "",
			TokenType: token.Type,
			Label:     "true_positive",
		})
	}

	return results, nil
}
