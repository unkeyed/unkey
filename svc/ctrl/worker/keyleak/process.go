package keyleak

import (
	"context"
	"time"

	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/restate/restateutil"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

func (s *Service) Process(ctx restate.Context, req *hydrav1.KeyLeakReport) (*hydrav1.KeyLeakReportResult, error) {
	hashes := make([]string, 0, len(req.GetTokens()))
	seen := make(map[string]bool, len(req.GetTokens()))
	for _, token := range req.GetTokens() {
		if !seen[token.GetHash()] {
			seen[token.GetHash()] = true
			hashes = append(hashes, token.GetHash())
		}
	}
	if len(hashes) == 0 {
		return &hydrav1.KeyLeakReportResult{}, nil
	}

	now, err := restateutil.Now(ctx)
	if err != nil {
		return nil, err
	}
	keys, err := restate.Run(ctx, func(rc restate.RunContext) ([]db.FindLeakedKeysByHashesRow, error) {
		queryCtx, cancel := context.WithTimeout(rc, 10*time.Second)
		defer cancel()

		return s.db.FindLeakedKeysByHashes(queryCtx, db.FindLeakedKeysByHashesParams{Hashes: hashes})
	}, runOptions("find leaked keys")...)
	if err != nil {
		return nil, err
	}

	byHash := make(map[string]db.FindLeakedKeysByHashesRow, len(keys))
	for _, key := range keys {
		byHash[key.Hash] = key
	}
	result := &hydrav1.KeyLeakReportResult{MatchedHashes: make([]string, 0, len(keys))}
	client := hydrav1.NewKeyLeakServiceClient(ctx)
	for _, token := range req.GetTokens() {
		key, found := byHash[token.GetHash()]
		if !found {
			continue
		}
		delete(byHash, token.GetHash())
		result.MatchedHashes = append(result.MatchedHashes, key.Hash)
		client.Notify().Send(&hydrav1.KeyLeakNotification{
			KeyId:         key.ID,
			OrgId:         key.OrgID,
			WorkspaceName: key.WorkspaceName,
			Source:        token.GetSource(),
			Url:           token.GetUrl(),
			Type:          token.GetType(),
			Date:          now.UTC().Format("Mon Jan 02 2006"),
		})
	}

	return result, nil
}
