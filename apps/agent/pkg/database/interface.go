package database

import (
	"context"

	"github.com/unkeyed/unkey/apps/agent/pkg/entities"
)

type Database interface {

	// Workspace
	InsertWorkspace(ctx context.Context, newWorkspace entities.Workspace) error
	UpdateWorkspace(ctx context.Context, workspace entities.Workspace) error
	FindWorkspace(ctx context.Context, workspaceId string) (entities.Workspace, bool, error)

	// KeyAuth
	InsertKeyAuth(ctx context.Context, newKeyAuth entities.KeyAuth) error
	DeleteKeyAuth(ctx context.Context, keyAuthId string) error
	FindKeyAuth(ctx context.Context, keyAuthId string) (keyauth entities.KeyAuth, found bool, err error)

	// Api
	InsertApi(ctx context.Context, api entities.Api) error
	FindApi(ctx context.Context, apiId string) (api entities.Api, found bool, err error)
	DeleteApi(ctx context.Context, apiId string) error
	FindApiByKeyAuthId(ctx context.Context, keyAuthId string) (api entities.Api, found bool, err error)
	ListAllApis(ctx context.Context, limit int, offset int) ([]entities.Api, error)

	// Key
	InsertKey(ctx context.Context, newKey entities.Key) error
	FindKeyById(ctx context.Context, keyId string) (key entities.Key, found bool, err error)
	FindKeyByHash(ctx context.Context, hash string) (key entities.Key, found bool, err error)
	UpdateKey(ctx context.Context, key entities.Key) error
	DeleteKey(ctx context.Context, keyId string) error
	DecrementRemainingKeyUsage(ctx context.Context, keyId string) (key entities.Key, err error)
	CountKeys(ctx context.Context, keyAuthId string) (int64, error)
	ListKeys(ctx context.Context, keyAuthId string, ownerId string, limit int, offset int) ([]entities.Key, error)

	// Stuff
	Close() error
}
