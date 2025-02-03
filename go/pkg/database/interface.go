package database

import (
	"context"

	"github.com/unkeyed/unkey/go/pkg/entities"
)

type Database interface {

	// Workspace
	InsertWorkspace(ctx context.Context, workspace entities.Workspace) error
	FindWorkspaceByID(ctx context.Context, id string) (entities.Workspace, error)
	UpdateWorkspacePlan(ctx context.Context, id string, plan entities.WorkspacePlan) error
	UpdateWorkspaceEnabled(ctx context.Context, id string, enabled bool) error
	// UpdateWorkspace(ctx context.Context, workspace entities.Workspace) error
	// FindWorkspace(ctx context.Context, workspaceId string) (entities.Workspace, bool, error)
	DeleteWorkspace(ctx context.Context, id string, hardDelete bool) error

	// KeyAuth
	// InsertKeyAuth(ctx context.Context, newKeyAuth entities.KeyAuth) error
	// DeleteKeyAuth(ctx context.Context, keyAuthId string) error
	// FindKeyAuth(ctx context.Context, keyAuthId string) (keyauth entities.KeyAuth, found bool, err error)

	// // Api
	// InsertApi(ctx context.Context, api entities.Api) error
	// FindApi(ctx context.Context, apiId string) (api entities.Api, found bool, err error)
	// DeleteApi(ctx context.Context, apiId string) error
	// FindApiByKeyAuthId(ctx context.Context, keyAuthId string) (api entities.Api, found bool, err error)
	// ListAllApis(ctx context.Context, limit int, offset int) ([]entities.Api, error)

	// Key
	// InsertKey(ctx context.Context, newKey entities.Key) error
	FindKeyByID(ctx context.Context, keyId string) (key entities.Key, err error)
	FindKeyByHash(ctx context.Context, hash string) (key entities.Key, err error)
	FindKeyForVerification(ctx context.Context, hash string) (key entities.Key, err error)
	// UpdateKey(ctx context.Context, key entities.Key) error
	// SoftDeleteKey(ctx context.Context, keyId string) error
	// DecrementRemainingKeyUsage(ctx context.Context, keyId string) (key entities.Key, err error)
	// CountKeys(ctx context.Context, keyAuthId string) (int64, error)
	// ListKeys(ctx context.Context, keyAuthId string, ownerId string, limit int, offset int) ([]entities.Key, error)

	// Ratelimit Namespace
	InsertRatelimitNamespace(ctx context.Context, namespace entities.RatelimitNamespace) error
	FindRatelimitNamespaceByID(ctx context.Context, id string) (entities.RatelimitNamespace, error)
	FindRatelimitNamespaceByName(ctx context.Context, workspaceID string, name string) (entities.RatelimitNamespace, error)
	DeleteRatelimitNamespace(ctx context.Context, id string) error

	// Ratelimit Override
	InsertRatelimitOverride(ctx context.Context, ratelimitOverride entities.RatelimitOverride) error
	FindRatelimitOverrideByIdentifier(ctx context.Context, identifier string) (ratelimitOverride entities.RatelimitOverride, err error)
	UpdateRatelimitOverride(ctx context.Context, override entities.RatelimitOverride) error
	DeleteRatelimitOverride(ctx context.Context, id string) error

	// Stuff
	Close() error
}
