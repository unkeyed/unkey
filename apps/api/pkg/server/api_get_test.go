package server

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/chronark/unkey/apps/api/pkg/cache"
	"github.com/chronark/unkey/apps/api/pkg/database"
	"github.com/chronark/unkey/apps/api/pkg/entities"
	"github.com/chronark/unkey/apps/api/pkg/logging"
	"github.com/chronark/unkey/apps/api/pkg/testutil"
	"github.com/chronark/unkey/apps/api/pkg/tracing"
	"github.com/chronark/unkey/apps/api/pkg/uid"
	"github.com/stretchr/testify/require"
)

func TestGetApi_Exists(t *testing.T) {

	resources := testutil.SetupResources(t)

	db, err := database.New(database.Config{
		PrimaryUs: os.Getenv("DATABASE_DSN"),
	})
	require.NoError(t, err)

	srv := New(Config{
		Logger:   logging.NewNoopLogger(),
		Cache:    cache.NewInMemoryCache[entities.Key](),
		Database: db,
		Tracer:   tracing.NewNoop(),
	})

	req := httptest.NewRequest("GET", fmt.Sprintf("/v1/apis/%s", resources.UserApi.Id), nil)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", resources.UnkeyKey))

	res, err := srv.app.Test(req)
	require.NoError(t, err)
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	require.NoError(t, err)
	require.Equal(t, 200, res.StatusCode)

	successResponse := GetApiResponse{}
	err = json.Unmarshal(body, &successResponse)
	require.NoError(t, err)

	require.Equal(t, resources.UserApi.Id, successResponse.Id)
	require.Equal(t, resources.UserApi.Name, successResponse.Name)
	require.Equal(t, resources.UserApi.WorkspaceId, successResponse.WorkspaceId)

}

func TestGetApi_NotFound(t *testing.T) {

	resources := testutil.SetupResources(t)

	db, err := database.New(database.Config{
		PrimaryUs: os.Getenv("DATABASE_DSN"),
	})
	require.NoError(t, err)

	srv := New(Config{
		Logger:   logging.NewNoopLogger(),
		Cache:    cache.NewInMemoryCache[entities.Key](),
		Database: db,
		Tracer:   tracing.NewNoop(),
	})

	fakeApiId := uid.Api()

	req := httptest.NewRequest("GET", fmt.Sprintf("/v1/apis/%s", fakeApiId), nil)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", resources.UnkeyKey))

	res, err := srv.app.Test(req)
	require.NoError(t, err)
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	require.NoError(t, err)

	require.Equal(t, 404, res.StatusCode)

	errorResponse := ErrorResponse{}
	err = json.Unmarshal(body, &errorResponse)
	require.NoError(t, err)

	require.Equal(t, NOT_FOUND, errorResponse.Code)
	require.Equal(t, fmt.Sprintf("unable to find api: %s", fakeApiId), errorResponse.Error)

}
