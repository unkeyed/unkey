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

	// h is the result of whatever algorithm we should use.
	// The section below is expected to populate this and we can use it to look up a key in the db
	var h string

	switch migration.Algorithm {
	case db.KeyMigrationsAlgorithmGithubcomSeamapiPrefixedApiKey:
		{
			parts := strings.Split(rawKey, "_")
			err = assert.Equal(len(parts), 3, "Expected prefixed api keys to have 3 segments")
			if err != nil {
				return nil, emptyLog, fault.Wrap(
					err,
					fault.Code(codes.URN(codes.Auth.Authentication.Malformed.URN())),
					fault.Public("Invalid key format"),
				)
			}

			b := sha256.Sum256([]byte(parts[2]))
			h = hex.EncodeToString(b[:])
		}
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
			Start:              extractStart(rawKey),
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

// extractStart extracts the start value from a key, handling both prefixed and non-prefixed keys
func extractStart(key string) string {
	// Check if the key has a prefix (format: prefix_actualkey)
	parts := strings.Split(key, "_")

	// If there are 2 or more parts, it's a prefixed key
	if len(parts) >= 2 {
		// Extract the prefix
		prefix := parts[0]
		// Get the actual key part (everything after first underscore)
		actualKey := strings.Join(parts[1:], "_")

		if len(actualKey) >= 4 {
			return fmt.Sprintf("%s_%s", prefix, actualKey[:4])
		}

		// If actual key is shorter than 4 chars, use what we have
		// this should never happen, but just in case
		return fmt.Sprintf("%s_%s", prefix, actualKey)
	}

	// No prefix, just return first 4 characters
	if len(key) >= 4 {
		return key[:4]
	}

	return key
}
