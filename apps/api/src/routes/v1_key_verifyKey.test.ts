import { describe, expect, test } from "bun:test";

import { ErrorResponse } from "@/pkg/errors";
import { init } from "@/pkg/global";
import { sha256 } from "@/pkg/hash/sha256";
import { newApp } from "@/pkg/hono/app";
import { newId } from "@/pkg/id";
import { KeyV1 } from "@/pkg/keys/v1";
import { testEnv } from "@/pkg/testutil/env";
import { fetchRoute } from "@/pkg/testutil/request";
import { seed } from "@/pkg/testutil/seed";
import { schema } from "@unkey/db";
import {
  V1KeysVerifyKeyRequest,
  V1KeysVerifyKeyResponse,
  registerV1KeysVerifyKey,
} from "./v1_keys_verifyKey";

test("returns 200", async () => {
  const env = testEnv();
  // @ts-ignore
  init({ env });
  const app = newApp();
  registerV1KeysVerifyKey(app);

  const r = await seed(env);

  const key = new KeyV1({ prefix: "test", byteLength: 16 }).toString();
  await r.database.insert(schema.keys).values({
    id: newId("key"),
    keyAuthId: r.userKeyAuth.id,
    hash: await sha256(key),
    start: key.slice(0, 8),
    workspaceId: r.userWorkspace.id,
    createdAt: new Date(),
  });

  const res = await fetchRoute<V1KeysVerifyKeyRequest, V1KeysVerifyKeyResponse>(app, {
    method: "POST",
    url: "/v1/keys.verifyKey",
    headers: {
      "Content-Type": "application/json",
    },
    body: {
      key,
      apiId: r.userApi.id,
    },
  });

  expect(res.status).toEqual(200);
  expect(res.body.valid).toBeTrue();
});

describe("bad request", () => {
  test("returns 400", async () => {
    const env = testEnv();
    // @ts-ignore
    init({ env });
    const app = newApp();
    registerV1KeysVerifyKey(app);

    const r = await seed(env);

    const key = new KeyV1({ prefix: "test", byteLength: 16 }).toString();
    await r.database.insert(schema.keys).values({
      id: newId("key"),
      keyAuthId: r.userKeyAuth.id,
      hash: await sha256(key),
      start: key.slice(0, 8),
      workspaceId: r.userWorkspace.id,
      createdAt: new Date(),
    });

    const res = await fetchRoute<any, ErrorResponse>(app, {
      method: "POST",
      url: "/v1/keys.verifyKey",
      headers: {
        "Content-Type": "application/json",
      },
      body: {
        something: "else",
      },
    });

    expect(res.status).toEqual(400);
  });
});

describe("with temporary key", () => {
  test("returns valid", async () => {
    const env = testEnv();
    // @ts-ignore
    init({ env });
    const app = newApp();
    registerV1KeysVerifyKey(app);

    const r = await seed(env);

    const key = new KeyV1({ prefix: "test", byteLength: 16 }).toString();
    await r.database.insert(schema.keys).values({
      id: newId("key"),
      keyAuthId: r.userKeyAuth.id,
      hash: await sha256(key),
      start: key.slice(0, 8),
      workspaceId: r.userWorkspace.id,
      createdAt: new Date(),
      expires: new Date(Date.now() + 5000),
    });

    const res = await fetchRoute<V1KeysVerifyKeyRequest, V1KeysVerifyKeyResponse>(app, {
      method: "POST",
      url: "/v1/keys.verifyKey",
      headers: {
        "Content-Type": "application/json",
      },
      body: {
        key,
        apiId: r.userApi.id,
      },
    });
    expect(res.status).toEqual(200);
    expect(res.body.valid).toBeTrue();

    await new Promise((resolve) => setTimeout(resolve, 6000));
    const secondResponse = await fetchRoute<V1KeysVerifyKeyRequest, V1KeysVerifyKeyResponse>(app, {
      method: "POST",
      url: "/v1/keys.verifyKey",
      headers: {
        "Content-Type": "application/json",
      },
      body: {
        key,
        apiId: r.userApi.id,
      },
    });
    expect(secondResponse.status).toEqual(200);
    expect(secondResponse.body.valid).toBeFalse();
  });
});

describe("with ip whitelist", () => {
  describe("with valid ip", () => {
    test("returns valid", async () => {
      const env = testEnv();
      // @ts-ignore
      init({ env });
      const app = newApp();
      registerV1KeysVerifyKey(app);

      const r = await seed(env);

      const keyAuthId = newId("keyAuth");
      await r.database.insert(schema.keyAuth).values({
        id: keyAuthId,
        workspaceId: r.userWorkspace.id,
      });

      const apiId = newId("api");
      await r.database.insert(schema.apis).values({
        id: apiId,
        workspaceId: r.userWorkspace.id,
        name: "test",
        authType: "key",
        keyAuthId: keyAuthId,
        ipWhitelist: JSON.stringify(["100.100.100.100"]),
      });

      const key = new KeyV1({ prefix: "test", byteLength: 16 }).toString();
      await r.database.insert(schema.keys).values({
        id: newId("key"),
        keyAuthId: keyAuthId,
        hash: await sha256(key),
        start: key.slice(0, 8),
        workspaceId: r.userWorkspace.id,
        createdAt: new Date(),
      });

      const res = await fetchRoute<V1KeysVerifyKeyRequest, V1KeysVerifyKeyResponse>(app, {
        method: "POST",
        url: "/v1/keys.verifyKey",
        headers: {
          "Content-Type": "application/json",
          "True-Client-IP": "100.100.100.100",
        },
        body: {
          key,
          apiId,
        },
      });
      expect(res.status).toEqual(200);
      expect(res.body.valid).toBeTrue();
    });
  });
  describe("with invalid ip", () => {
    test("returns invalid", async () => {
      const env = testEnv();
      // @ts-ignore
      init({ env });
      const app = newApp();
      registerV1KeysVerifyKey(app);

      const r = await seed(env);

      const keyAuthid = newId("keyAuth");
      await r.database.insert(schema.keyAuth).values({
        id: keyAuthid,
        workspaceId: r.userWorkspace.id,
      });

      const apiId = newId("api");
      await r.database.insert(schema.apis).values({
        id: apiId,
        workspaceId: r.userWorkspace.id,
        name: "test",
        authType: "key",
        keyAuthId: keyAuthid,
        ipWhitelist: JSON.stringify(["100.100.100.100"]),
      });

      const key = new KeyV1({ prefix: "test", byteLength: 16 }).toString();
      await r.database.insert(schema.keys).values({
        id: newId("key"),
        keyAuthId: keyAuthid,
        hash: await sha256(key),
        start: key.slice(0, 8),
        workspaceId: r.userWorkspace.id,
        createdAt: new Date(),
      });

      const res = await fetchRoute<V1KeysVerifyKeyRequest, V1KeysVerifyKeyResponse>(app, {
        method: "POST",
        url: "/v1/keys.verifyKey",
        headers: {
          "Content-Type": "application/json",
          "True-Client-IP": "200.200.200.200",
        },
        body: {
          key,
          apiId: r.userApi.id,
        },
      });
      expect(res.status).toEqual(200);
      expect(res.body.valid).toBeFalse();
      expect(res.body.code).toEqual("FORBIDDEN");
    });
  });
});

// func TestVerifyKey_WithIpWhitelist_Blocked(t * testing.T) {
//   ctx:= context.Background()

//   resources:= testutil.SetupResources(t)

//   keyAuth:= entities.KeyAuth{
//     Id: uid.KeyAuth(),
//       WorkspaceId: resources.UserWorkspace.Id,
// 	}
//   err:= resources.Database.InsertKeyAuth(ctx, keyAuth)
//   require.NoError(t, err)

//   api:= entities.Api{
//     Id: uid.Api(),
//       KeyAuthId: keyAuth.Id,
//         AuthType: entities.AuthTypeKey,
//           Name: "test",
//             WorkspaceId: resources.UserWorkspace.Id,
//               IpWhitelist: []string{ "100.100.100.100" },
//   }
//   err = resources.Database.InsertApi(ctx, api)
//   require.NoError(t, err)

//   key:= uid.New(16, "test")
//   err = resources.Database.InsertKey(ctx, entities.Key{
//     Id: uid.Key(),
//     KeyAuthId: keyAuth.Id,
//     WorkspaceId: api.WorkspaceId,
//     Hash: hash.Sha256(key),
//     CreatedAt: time.Now(),
//   })
//   require.NoError(t, err)

//   srv:= New(Config{
//     Logger: logging.NewNoopLogger(),
//     KeyCache: cache.NewNoopCache[entities.Key](),
//     ApiCache: cache.NewNoopCache[entities.Api](),
//     Database: resources.Database,
//     Tracer: tracing.NewNoop(),
//   })

//   buf:= bytes.NewBufferString(fmt.Sprintf(`{
// 		"key":"%s"
// 		}`, key))

//   req:= httptest.NewRequest("POST", "/v1/keys.verifyKey", buf)
//   req.Header.Set("Content-Type", "application/json")
//   req.Header.Set("Fly-Client-IP", "1.2.3.4")

//   res, err := srv.app.Test(req)
//   require.NoError(t, err)
// 	defer res.Body.Close()

//   body, err := io.ReadAll(res.Body)
//   require.NoError(t, err)
//   require.Equal(t, 200, res.StatusCode)

//   verifyKeyResponse:= VerifyKeyResponseV1{ }
//   err = json.Unmarshal(body, & verifyKeyResponse)
//   require.NoError(t, err)

//   require.Equal(t, errors.FORBIDDEN, verifyKeyResponse.Code)

// }

// func TestVerifyKey_WithRemaining(t * testing.T) {
//   ctx:= context.Background()

//   resources:= testutil.SetupResources(t)

//   key:= uid.New(16, "test")
//   remaining:= int32(10)
//   err:= resources.Database.InsertKey(ctx, entities.Key{
//     Id: uid.Key(),
//     KeyAuthId: resources.UserKeyAuth.Id,
//     WorkspaceId: resources.UserWorkspace.Id,
//     Hash: hash.Sha256(key),
//     CreatedAt: time.Now(),
//     Remaining:   & remaining,
//   })
//   require.NoError(t, err)

//   srv:= New(Config{
//     Logger: logging.NewNoopLogger(),
//     KeyCache: cache.NewNoopCache[entities.Key](),
//     ApiCache: cache.NewNoopCache[entities.Api](),
//     Database: resources.Database,
//     Tracer: tracing.NewNoop(),
//     Ratelimit: ratelimit.NewInMemory(),
//     Metrics: metrics.NewNoop(),
//   })

//   buf:= bytes.NewBufferString(fmt.Sprintf(`{
// 		"key":"%s"
// 		}`, key))

//   req:= httptest.NewRequest("POST", "/v1/keys.verifyKey", buf)
//   req.Header.Set("Content-Type", "application/json")

//   // Use up 10 requests
//   for i := 9; i >= 0; i-- {

//     res, err := srv.app.Test(req)
//     require.NoError(t, err)
// 		defer res.Body.Close()

//     body1, err := io.ReadAll(res.Body)
//     require.NoError(t, err)
//     require.Equal(t, 200, res.StatusCode)

//     vr:= VerifyKeyResponseV1{ }
//     err = json.Unmarshal(body1, & vr)
//     require.NoError(t, err)

//     require.True(t, vr.Valid)
//     require.NotNil(t, vr.Remaining)
//     require.Equal(t, int32(i), * vr.Remaining)
//   }

//   // now it should be all used up and no longer valid

//   res2, err := srv.app.Test(req)
//   require.NoError(t, err)
// 	defer res2.Body.Close()

//   body2, err := io.ReadAll(res2.Body)
//   require.NoError(t, err)
//   require.Equal(t, 200, res2.StatusCode)

//   verifyRes2:= VerifyKeyResponseV1{ }
//   err = json.Unmarshal(body2, & verifyRes2)
//   require.NoError(t, err)

//   require.False(t, verifyRes2.Valid)
//   require.Equal(t, int32(0), * verifyRes2.Remaining)

// }

// type mockAnalytics struct {
// 	calledPublish atomic.Int32
// }

// func(m * mockAnalytics) PublishKeyVerificationEvent(ctx context.Context, event analytics.KeyVerificationEvent) {
//   m.calledPublish.Add(1)
// }
// func(m * mockAnalytics) GetKeyStats(ctx context.Context, keyId string)(analytics.KeyStats, error) {
//   return analytics.KeyStats{ }, fmt.Errorf("Implement me")
// }

// func TestVerifyKey_ShouldReportUsageWhenUsageExceeded(t * testing.T) {
//   t.Parallel()
//   ctx:= context.Background()

//   resources:= testutil.SetupResources(t)

//   key:= uid.New(16, "test")
//   err:= resources.Database.InsertKey(ctx, entities.Key{
//     Id: uid.Key(),
//     KeyAuthId: resources.UserKeyAuth.Id,
//     WorkspaceId: resources.UserWorkspace.Id,
//     Hash: hash.Sha256(key),
//     CreatedAt: time.Now(),
//     Remaining: util.Pointer(int32(0)),
//   })
//   require.NoError(t, err)

//   a:= & mockAnalytics{ }
//   srv:= New(Config{
//     Logger: logging.NewNoopLogger(),
//     KeyCache: cache.NewNoopCache[entities.Key](),
//     ApiCache: cache.NewNoopCache[entities.Api](),
//     Database: resources.Database,
//     Tracer: tracing.NewNoop(),
//     Analytics: a,
//   })

//   buf:= bytes.NewBufferString(fmt.Sprintf(`{
// 		"key":"%s"
// 		}`, key))

//   req:= httptest.NewRequest("POST", "/v1/keys.verifyKey", buf)
//   req.Header.Set("Content-Type", "application/json")

//   res, err := srv.app.Test(req)
//   require.NoError(t, err)
// 	defer res.Body.Close()

//   body, err := io.ReadAll(res.Body)
//   require.NoError(t, err)

//   require.Equal(t, 200, res.StatusCode)

//   successResponse:= VerifyKeyResponseV1{ }
//   err = json.Unmarshal(body, & successResponse)
//   require.NoError(t, err)

//   require.False(t, successResponse.Valid)
//   require.Equal(t, int32(1), a.calledPublish.Load())

// }
