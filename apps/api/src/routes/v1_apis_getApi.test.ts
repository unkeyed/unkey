

import { describe, test, expect, beforeAll, afterAll } from "bun:test"
import { testClient } from 'hono/testing'
import { registerV1ApisGetApi } from "./v1_apis_getApi"
import { app } from "@/pkg/hono/app"
import { init } from "@/pkg/global"
import { Env } from "@/pkg/env"
import { createMysqlDatabase } from "@/pkg/testutil/database"
import { StartedMySqlContainer } from "@testcontainers/mysql"


let mysql: StartedMySqlContainer

beforeAll(async () => {

  mysql = await createMysqlDatabase()
  process.env.DATABASE_HOST = mysql.getHost()
  process.env.DATABASE_USERNAME = mysql.getUsername()
  process.env.DATABASE_PASSWORD = mysql.getUserPassword()

  init({ env: process.env as unknown as Env["Bindings"] })
  registerV1ApisGetApi(app)
})

afterAll(async () => {
  await mysql.stop()
})

describe("when api exists", () => {
  test("return api", async () => {

    const client = testClient<ReturnType<typeof registerV1ApisGetApi>>(app)
    const res = await client.v1["apis.getApi"].$get({
      query: {
        apiId: "api_does_not_exist",
      }
    }, {
      headers: {
        Authorization: "Bearer x"
      }
    })
    console.log(await res.json())
    expect(res.status).toEqual(404)

  })
})

// TestGetApi_Exists(t * testing.T) {
//   t.Parallel()

//   resources:= testutil.SetupResources(t)

//   srv:= New(Config{
//     Logger: logging.NewNoopLogger(),
//     KeyCache: cache.NewNoopCache[entities.Key](),
//     ApiCache: cache.NewNoopCache[entities.Api](),
//     Database: resources.Database,
//     Tracer: tracing.NewNoop(),
//   })

//   req:= httptest.NewRequest("GET", fmt.Sprintf("/v1/apis/%s", resources.UserApi.Id), nil)
//   req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", resources.UserRootKey))

//   res, err := srv.app.Test(req)
//   require.NoError(t, err)
// 	defer res.Body.Close()

//   body, err := io.ReadAll(res.Body)
//   require.NoError(t, err)
//   require.Equal(t, 200, res.StatusCode)

//   successResponse:= GetApiResponse{ }
//   err = json.Unmarshal(body, & successResponse)
//   require.NoError(t, err)

//   require.Equal(t, resources.UserApi.Id, successResponse.Id)
//   require.Equal(t, resources.UserApi.Name, successResponse.Name)
//   require.Equal(t, resources.UserApi.WorkspaceId, successResponse.WorkspaceId)

// }

// func TestGetApi_NotFound(t * testing.T) {
//   t.Parallel()
//   resources:= testutil.SetupResources(t)

//   srv:= New(Config{
//     Logger: logging.NewNoopLogger(),
//     KeyCache: cache.NewNoopCache[entities.Key](),
//     ApiCache: cache.NewNoopCache[entities.Api](),
//     Database: resources.Database,
//     Tracer: tracing.NewNoop(),
//   })

//   fakeApiId:= uid.Api()

//   req:= httptest.NewRequest("GET", fmt.Sprintf("/v1/apis/%s", fakeApiId), nil)
//   req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", resources.UserRootKey))

//   res, err := srv.app.Test(req)
//   require.NoError(t, err)
// 	defer res.Body.Close()

//   body, err := io.ReadAll(res.Body)
//   require.NoError(t, err)

//   require.Equal(t, 404, res.StatusCode)

//   errorResponse:= errors.ErrorResponse{ }
//   err = json.Unmarshal(body, & errorResponse)
//   require.NoError(t, err)

//   require.Equal(t, errors.NOT_FOUND, errorResponse.Error.Code)
//   require.Equal(t, fmt.Sprintf("unable to find api: %s", fakeApiId), errorResponse.Error.Message)

// }

// func TestGetApi_WithIpWhitelist(t * testing.T) {
//   t.Parallel()
//   ctx:= context.Background()
//   resources:= testutil.SetupResources(t)

//   srv:= New(Config{
//     Logger: logging.NewNoopLogger(),
//     KeyCache: cache.NewNoopCache[entities.Key](),
//     ApiCache: cache.NewNoopCache[entities.Api](),
//     Database: resources.Database,
//     Tracer: tracing.NewNoop(),
//   })

//   keyAuth:= entities.KeyAuth{
//     Id: uid.KeyAuth(),
//       WorkspaceId: resources.UserWorkspace.Id,
// 	}
//   err:= resources.Database.InsertKeyAuth(ctx, keyAuth)
//   require.NoError(t, err)

//   api:= entities.Api{
//     Id: uid.Api(),
//       Name: "test",
//         WorkspaceId: resources.UserWorkspace.Id,
//           IpWhitelist: []string{ "127.0.0.1", "1.1.1.1" },
//     AuthType: entities.AuthTypeKey,
//       KeyAuthId: keyAuth.Id,
// 	}

//   err = resources.Database.InsertApi(ctx, api)
//   require.NoError(t, err)

//   req:= httptest.NewRequest("GET", fmt.Sprintf("/v1/apis/%s", api.Id), nil)
//   req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", resources.UserRootKey))

//   res, err := srv.app.Test(req)
//   require.NoError(t, err)
// 	defer res.Body.Close()

//   body, err := io.ReadAll(res.Body)
//   require.NoError(t, err)
//   require.Equal(t, 200, res.StatusCode)

//   successResponse:= GetApiResponse{ }
//   err = json.Unmarshal(body, & successResponse)
//   require.NoError(t, err)

//   require.Equal(t, api.Id, successResponse.Id)
//   require.Equal(t, api.Name, successResponse.Name)
//   require.Equal(t, api.WorkspaceId, successResponse.WorkspaceId)
//   require.Equal(t, api.IpWhitelist, successResponse.IpWhitelist)

// }
