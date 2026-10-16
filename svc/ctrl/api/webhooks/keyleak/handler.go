package keyleak

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	restate "github.com/restatedev/sdk-go"
	"github.com/restatedev/sdk-go/ingress"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/hash"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/webhook"
	"github.com/unkeyed/unkey/pkg/webhook/verifiers/githubsecrets"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

const bodyBytesMax = 4 << 20

type handler struct {
	db      db.Database
	restate *ingress.Client
	clock   clock.Clock
}

func New(database db.Database, client *ingress.Client, verifier webhook.Verifier, clock clock.Clock) http.Handler {
	h := &handler{db: database, restate: client, clock: clock}
	receiver := webhook.New("github_secret_scanning", verifier, webhook.WithMaxBodySize(bodyBytesMax)).
		OnResponse([]string{githubsecrets.EventType}, h.report)
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

func (h *handler) report(ctx context.Context, event webhook.Event) (webhook.Response, error) {
	var tokens []reportedToken
	if err := json.Unmarshal(event.Payload, &tokens); err != nil || tokens == nil {
		return webhook.Response{}, webhook.ErrBadRequest
	}
	for _, token := range tokens {
		if token.Token == "" || token.Type == "" || token.Source == "" {
			return webhook.Response{}, webhook.ErrBadRequest
		}
	}
	results, err := h.classifyAndEnqueue(ctx, tokens)
	if err != nil {
		return webhook.Response{}, fmt.Errorf("%w: %w", webhook.ErrUnavailable, err)
	}
	body, err := json.Marshal(results)
	if err != nil {
		return webhook.Response{}, err
	}
	return webhook.Response{StatusCode: http.StatusCreated, ContentType: "application/json", Body: body}, nil
}

func (h *handler) classifyAndEnqueue(ctx context.Context, tokens []reportedToken) ([]result, error) {
	results := make([]result, 0, len(tokens))
	if len(tokens) == 0 {
		return results, nil
	}
	hashes := make([]string, 0, len(tokens))
	seen := make(map[string]bool, len(tokens))
	for _, token := range tokens {
		hashed := hash.Sha256(token.Token)
		if !seen[hashed] {
			seen[hashed] = true
			hashes = append(hashes, hashed)
		}
	}
	keys, err := h.db.FindLeakedKeysByHashes(ctx, hashes)
	if err != nil {
		return nil, fmt.Errorf("find leaked keys: %w", err)
	}
	byHash := make(map[string]db.FindLeakedKeysByHashesRow, len(keys))
	for _, key := range keys {
		byHash[key.Hash] = key
	}
	enqueued := make(map[string]bool, len(keys))
	for _, token := range tokens {
		hashed := hash.Sha256(token.Token)
		key, found := byHash[hashed]
		if !found {
			results = append(results, result{
				TokenRaw:  "",
				TokenHash: hashed,
				TokenType: token.Type,
				Label:     "false_positive",
			})
			continue
		}
		if !enqueued[key.ID] {
			_, err := hydrav1.NewKeyLeakServiceIngressClient(h.restate).Notify().Send(ctx, &hydrav1.KeyLeakNotification{
				KeyId:         key.ID,
				OrgId:         key.OrgID,
				WorkspaceName: key.WorkspaceName,
				Source:        token.Source,
				Url:           token.URL,
				Type:          token.Type,
				Date:          h.clock.Now().UTC().Format("Mon Jan 02 2006"),
			}, restate.WithIdempotencyKey("leaked-key:"+key.ID))
			if err != nil {
				return nil, fmt.Errorf("enqueue key leak notification: %w", err)
			}
			enqueued[key.ID] = true
		}
		results = append(results, result{
			TokenRaw:  token.Token,
			TokenHash: "",
			TokenType: token.Type,
			Label:     "true_positive",
		})
	}
	logger.Info("Accepted key leak report", "reported", len(tokens), "matched", len(keys))
	return results, nil
}
