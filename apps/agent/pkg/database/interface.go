package database

import (
	"context"

	apisv1 "github.com/unkeyed/unkey/apps/agent/gen/proto/apis/v1"
	authenticationv1 "github.com/unkeyed/unkey/apps/agent/gen/proto/authentication/v1"
	workspacesv1 "github.com/unkeyed/unkey/apps/agent/gen/proto/workspaces/v1"
)

type Database interface {

	// Workspace
	InsertWorkspace(ctx context.Context, newWorkspace *workspacesv1.Workspace) error
	UpdateWorkspace(ctx context.Context, workspace *workspacesv1.Workspace) error
	FindWorkspace(ctx context.Context, workspaceId string) (*workspacesv1.Workspace, bool, error)
	DeleteWorkspace(ctx context.Context, workspaceId string) error

	// KeyAuth
	InsertKeyAuth(ctx context.Context, newKeyAuth *authenticationv1.KeyAuth) error
	DeleteKeyAuth(ctx context.Context, keyAuthId string) error
	FindKeyAuth(ctx context.Context, keyAuthId string) (keyauth *authenticationv1.KeyAuth, found bool, err error)

	// Api
	InsertApi(ctx context.Context, api *apisv1.Api) error
	FindApi(ctx context.Context, apiId string) (api *apisv1.Api, found bool, err error)
	DeleteApi(ctx context.Context, apiId string) error
	FindApiByKeyAuthId(ctx context.Context, keyAuthId string) (api *apisv1.Api, found bool, err error)
	ListAllApis(ctx context.Context, limit int, offset int) ([]*apisv1.Api, error)

	// Key
	InsertKey(ctx context.Context, newKey *authenticationv1.Key) error
	FindKeyById(ctx context.Context, keyId string) (key *authenticationv1.Key, found bool, err error)
	FindKeyByHash(ctx context.Context, hash string) (key *authenticationv1.Key, found bool, err error)
	UpdateKey(ctx context.Context, key *authenticationv1.Key) error
	SoftDeleteKey(ctx context.Context, keyId string) error
	DecrementRemainingKeyUsage(ctx context.Context, keyId string) (key *authenticationv1.Key, err error)
	CountKeys(ctx context.Context, keyAuthId string) (int64, error)
	ListKeys(ctx context.Context, keyAuthId string, ownerId string, limit int, offset int) ([]*authenticationv1.Key, error)

	// Stuff
	Close() error
}
