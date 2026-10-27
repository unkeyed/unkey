package keyleak

import (
	"context"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"time"

	restate "github.com/restatedev/sdk-go"
	"github.com/restatedev/sdk-go/ingress"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/hash"
	"github.com/unkeyed/unkey/pkg/webhook"
	"github.com/unkeyed/unkey/pkg/webhook/verifiers/githubsecrets"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

const bodyBytesMax = 4 << 20

type handler struct {
	restate *ingress.Client
	db      db.Database
}

type Config struct {
	Restate  *ingress.Client
	Database db.Database
	Verifier webhook.Verifier
}

func New(cfg Config) http.Handler {
	h := &handler{restate: cfg.Restate, db: cfg.Database}

	return webhook.New("github_secret_scanning", cfg.Verifier,
		webhook.WithMaxBodySize(bodyBytesMax),
		webhook.WithTimeout(10*time.Second),
	).On([]string{githubsecrets.EventType}, h.report)
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

	hashes := make([]string, 0, len(tokens))
	byHash := make(map[string]reportedToken, len(tokens))
	for _, token := range tokens {
		digest := hash.Sha256(token.Token)
		if _, found := byHash[digest]; found {
			continue
		}

		byHash[digest] = token
		hashes = append(hashes, digest)
	}

	keys, err := h.db.FindLeakedKeysByHashes(ctx, db.FindLeakedKeysByHashesParams{Hashes: hashes})
	if err != nil {
		return fmt.Errorf("%w: find leaked keys: %w", webhook.ErrUnavailable, err)
	}
	if len(keys) == 0 {
		return nil
	}

	reportID := hash.Sha256(string(event.Payload))
	date := time.Now().UTC().Format("Mon Jan 02 2006")
	matches := make([]*hydrav1.KeyLeakMatch, 0, len(keys))
	for _, key := range keys {
		token, found := byHash[key.Hash]
		if !found {
			continue
		}

		delete(byHash, key.Hash)
		matches = append(matches, &hydrav1.KeyLeakMatch{
			Hash: key.Hash, WorkspaceId: key.WorkspaceID, RootKey: key.KeyType == "rootKey",
			Notification: &hydrav1.KeyLeakNotification{
				KeyId: key.ID, OrgId: key.OrgID, WorkspaceName: key.WorkspaceName,
				Source: token.Source, Url: token.URL, Type: token.Type, Date: date,
			},
		})
	}
	slices.SortFunc(matches, func(a, b *hydrav1.KeyLeakMatch) int { return strings.Compare(a.GetHash(), b.GetHash()) })

	client := hydrav1.NewKeyLeakServiceIngressClient(h.restate)
	for batch := range slices.Chunk(matches, 100) {
		hashes := make([]string, 0, len(batch))
		for _, key := range batch {
			hashes = append(hashes, key.GetHash())
		}

		_, err := client.NotifyBatch().Send(ctx, &hydrav1.KeyLeakBatchRequest{ReportId: reportID, Keys: batch},
			restate.WithIdempotencyKey(reportID+":"+hash.Sha256(strings.Join(hashes, ","))))
		if err != nil {
			return fmt.Errorf("%w: persist key leak batch: %w", webhook.ErrUnavailable, err)
		}
	}

	return nil
}
