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
			parts := strings.SplitN(rawKey, "_", 3)
			err = assert.Equal(len(parts), 3, "Expected prefixed api keys to have 3 segments")
			if err != nil {
				return nil, emptyLog, fault.Wrap(
					err,
					fault.Code(codes.URN(codes.Auth.Authentication.Malformed.URN())),
					fault.Public("Invalid key format"),
				)
			}

			// Hash the long token for lookup
			b := sha256.Sum256([]byte(parts[2]))
			h = hex.EncodeToString(b[:])

			// Extract start using the already-parsed parts
			// Format: prefix_shortToken_longToken -> prefix_shor (first 4 chars of shortToken_longToken)
			prefix := parts[0]                        // Can contain underscores (e.g., "my_company")
			actualKey := parts[1] + "_" + parts[2]    // shortToken_longToken

			if len(actualKey) >= 4 {
				start = fmt.Sprintf("%s_%s", prefix, actualKey[:4])
			} else {
				start = fmt.Sprintf("%s_%s", prefix, actualKey)
			}
		}
	case db.KeyMigrationsAlgorithmSha256:
		return nil, emptyLog, fault.New(
			"sha256 doesn't require a migration",
			fault.Public("We could not find the requested key"),
		)
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
