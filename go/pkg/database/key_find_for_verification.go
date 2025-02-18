package database

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"github.com/unkeyed/unkey/go/pkg/database/transform"
	"github.com/unkeyed/unkey/go/pkg/entities"
	"github.com/unkeyed/unkey/go/pkg/fault"
)

func (db *database) FindKeyForVerification(ctx context.Context, hash string) (entities.Key, error) {

	res, err := db.read().FindKeyForVerification(ctx, hash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return entities.Key{}, fault.Wrap(err,
				fault.WithTag(fault.NOT_FOUND),
				fault.WithDesc("key not found", "The key does not exist."),
			)
		}

		return entities.Key{}, fault.Wrap(err,
			fault.WithTag(fault.DATABASE_ERROR),
		)
	}

	key, err := transform.KeyModelToEntity(res.Key)
	if err != nil {
		return entities.Key{}, err
	}

	key.Permissions = []string{}
	if res.Permissions.Valid {
		key.Permissions = strings.Split(res.Permissions.String, ",")
	}

	identity, err := transform.IdentityModelToEntity(res.Identity)
	if err != nil {
		return entities.Key{}, err
	}
	key.Identity = &identity

	return key, nil
}
