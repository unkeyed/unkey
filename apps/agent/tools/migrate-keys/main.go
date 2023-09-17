package main

import (
	"context"
	"log"
	"time"

	"github.com/unkeyed/unkey/apps/agent/pkg/database"
	"github.com/unkeyed/unkey/apps/agent/pkg/entities"
	"github.com/unkeyed/unkey/apps/agent/pkg/hash"
	"github.com/unkeyed/unkey/apps/agent/pkg/uid"
)

type externalKey struct {
	value   string
	ownerId string
	name    string
}

const (
	DATABASE_URL = ""
	WORKSPACE_ID = ""
	API_ID       = ""
)

// TODO
var keys = []externalKey{}

func main() {

	ctx := context.Background()
	db, err := database.New(database.Config{
		PrimaryUs: DATABASE_URL,
	})
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	_, found, err := db.FindWorkspace(ctx, WORKSPACE_ID)
	if err != nil {
		log.Fatal(err)
	}
	if !found {
		log.Fatal("workspace not found")
	}

	api, found, err := db.FindApi(ctx, API_ID)
	if err != nil {
		log.Fatal(err)
	}
	if !found {
		log.Fatal("api not found")
	}

	if api.AuthType != entities.AuthTypeKey || api.KeyAuthId == "" {
		log.Fatal("api is not setup to handle api keys")

	}

	for _, key := range keys {
		log.Printf("migrating key %s", key.value)
		// how many chars to store, this includes the prefix, delimiter and the first 4 characters of the key
		startLength := 4
		keyHash := hash.Sha256(key.value)

		newKey := entities.Key{
			Id:          uid.Key(),
			KeyAuthId:   api.KeyAuthId,
			WorkspaceId: WORKSPACE_ID,
			Name:        key.name,
			Hash:        keyHash,
			Start:       key.value[:startLength],
			OwnerId:     key.ownerId,
			// Meta:        req.Meta,
			CreatedAt: time.Now(),
		}
		// if req.Expires > 0 {
		// 	newKey.Expires = time.UnixMilli(req.Expires)
		// }
		// if req.Remaining > 0 {
		// 	remaining := req.Remaining
		// 	newKey.Remaining = &remaining
		// }
		// if req.Ratelimit != nil {
		// 	newKey.Ratelimit = &entities.Ratelimit{
		// 		Type:           req.Ratelimit.Type,
		// 		Limit:          req.Ratelimit.Limit,
		// 		RefillRate:     req.Ratelimit.RefillRate,
		// 		RefillInterval: req.Ratelimit.RefillInterval,
		// 	}
		// }

		err = db.CreateKey(ctx, newKey)
		if err != nil {
			log.Fatalf("unable to store key: %s", err.Error())
		}
	}

}
