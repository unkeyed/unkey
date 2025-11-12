package keys

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/unkeyed/unkey/go/pkg/assert"
	"github.com/unkeyed/unkey/go/pkg/codes"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/fault"
	"github.com/unkeyed/unkey/go/pkg/hash"
	"github.com/unkeyed/unkey/go/pkg/otel/tracing"
	"github.com/unkeyed/unkey/go/pkg/zen"
)

// GetMigrated uses a special hashing algorithm to retrieve a key from the database and
// migrates the key to our own hashing algorithm.
func (s *service) GetMigrated(ctx context.Context, sess *zen.Session, rawKey string, migrationID string) (*KeyVerifier, func(), error) {
	ctx, span := tracing.Start(ctx, "keys.GetMigrated")
	defer span.End()

	err := assert.NotEmpty(rawKey)
	if err != nil {
		return nil, emptyLog, fault.Wrap(err, fault.Internal("rawKey is empty"))
	}

	migration, err := db.Query.FindKeyMigrationByID(ctx, s.db.RO(), db.FindKeyMigrationByIDParams{
		ID:          migrationID,
		WorkspaceID: sess.AuthorizedWorkspaceID(),
	})
	if err != nil {
		if db.IsNotFound(err) {
			// nolint:exhaustruct
			return &KeyVerifier{
				Status:  StatusNotFound,
				message: "migration does not exist",
			}, emptyLog, nil
		}

		return nil, emptyLog, fault.Wrap(
			err,
			fault.Internal("unable to load migration"),
			fault.Public("We could not load the requested migration."),
		)
	}

	// h is the hash result for the algorithm-specific lookup
	// start is the prefix + first few chars for display/indexing
	// Each algorithm is responsible for computing both values based on its key format
	var h string
	var start string

	switch migration.Algorithm {
	case db.KeyMigrationsAlgorithmGithubcomSeamapiPrefixedApiKey:
		{
			// Parse from the right since shortToken and longToken never contain underscores,
			// but prefix can (e.g., "my_company_shortToken_longToken")
			parts := strings.Split(rawKey, "_")
			if len(parts) < 3 {
				return nil, emptyLog, fault.Wrap(
					fmt.Errorf("expected at least 3 segments, got %d", len(parts)),
					fault.Code(codes.URN(codes.Auth.Authentication.Malformed.URN())),
					fault.Public("Invalid key format"),
				)
			}

			// Take last 2 parts as shortToken and longToken
			longToken := parts[len(parts)-1]
			shortToken := parts[len(parts)-2]
			// Everything before is the prefix (may contain underscores)
			prefix := strings.Join(parts[:len(parts)-2], "_")

			// Hash the long token for lookup
			b := sha256.Sum256([]byte(longToken))
			h = hex.EncodeToString(b[:])

			// Extract start using the already-parsed parts
			// Format: prefix_shortToken_longToken -> prefix_shor (first 4 chars of shortToken_longToken)
			actualKey := shortToken + "_" + longToken

			if len(actualKey) >= 4 {
				start = fmt.Sprintf("%s_%s", prefix, actualKey[:4])
			} else {
				start = fmt.Sprintf("%s_%s", prefix, actualKey)
			}
		}
	case db.KeyMigrationsAlgorithmSha256:
		// If we have a sha256 already migrated key and we didn't find it in the first place
		// then it doesn't exist, and there is nothing to migrate here.
		return &KeyVerifier{
			Status:  StatusNotFound,
			message: "key does not exist",
		}, emptyLog, nil
	default:
		return nil, emptyLog, fault.New(
			fmt.Sprintf("unsupported migration algorithm %s", migration.Algorithm),
			fault.Public(fmt.Sprintf("We could not load the requested migration for algorithm %s.", migration.Algorithm)),
		)
	}

	key, log, err := s.Get(ctx, sess, h)
	if err != nil {
		return nil, log, err
	}

	if key.Key.PendingMigrationID.Valid && key.Key.PendingMigrationID.String == migrationID {
		newHash := hash.Sha256(rawKey)
		err = db.Query.UpdateKeyHashAndMigration(ctx, s.db.RW(), db.UpdateKeyHashAndMigrationParams{
			ID:                 key.Key.ID,
			Hash:               newHash,
			Start:              start,
			PendingMigrationID: sql.NullString{Valid: false, String: ""},
			UpdatedAtM:         sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
		})
		if err != nil {
			return nil, log, fault.Wrap(
				err,
				fault.Code(codes.App.Internal.UnexpectedError.URN()),
				fault.Public("We could not update the key hash and migration id"),
			)
		}

		s.keyCache.Remove(
			ctx,
			h,
			newHash,
		)
	}

	return key, log, nil
}
