import { test, expect } from "bun:test";
import { newHonoApp } from "@/pkg/hono/app";

import { init } from "@/pkg/global";
import { testEnv } from "@/pkg/testutil/env";
import { seed } from "@/pkg/testutil/setup";
import { newId } from "@/pkg/id";
import { sha256 } from "@/pkg/hash/sha256";
import { KeyV1 } from "@/pkg/keys/v1";
import { V1KeysGetKeyResponse, registerV1KeysGetKey } from "./v1_keys_getKey";
import { schema } from "@unkey/db";
import { fetchRoute } from "@/pkg/testutil/request";

test("returns 200", async () => {
  const env = testEnv();
  // @ts-ignore
  init({ env });

  const app = newHonoApp();
  registerV1KeysGetKey(app);

  const r = await seed(env);

  const key = {
    id: newId("key"),
    keyAuthId: r.userKeyAuth.id,
    workspaceId: r.userWorkspace.id,
    start: "test",
    name: "test",
    hash: await sha256(new KeyV1({ byteLength: 16 }).toString()),
    createdAt: new Date(),
  };
  await r.database.insert(schema.keys).values(key);

  // require.Equal(t, key.Id, successResponse.Id)
  // require.Equal(t, resources.UserApi.Id, successResponse.ApiId)
  // require.Equal(t, key.WorkspaceId, successResponse.WorkspaceId)
  // require.Equal(t, key.Name, successResponse.Name)
  // require.True(t, strings.HasPrefix(key.Hash, successResponse.Start))
  // require.WithinDuration(t, key.CreatedAt, time.UnixMilli(successResponse.CreatedAt), time.Second)

  const res = await fetchRoute<never, V1KeysGetKeyResponse>(app, {
    method: "GET",
    url: `/v1/keys.getKey?keyId=${key.id}`,
    headers: {
      Authorization: `Bearer ${r.rootKey}`,
    },
  });

  expect(res.status).toEqual(200);

  console.log(res);
  expect(res.body.id).toEqual(key.id);
  expect(res.body.apiId).toEqual(r.userApi.id);
  expect(res.body.workspaceId).toEqual(key.workspaceId);
  expect(res.body.name).toEqual(key.name);
  expect(res.body.start).toEqual(key.start);
  expect(res.body.createdAt).toEqual(key.createdAt.getTime());
});
