import { expect, test } from "bun:test";

import { init } from "@/pkg/global";
import { sha256 } from "@/pkg/hash/sha256";
import { newApp } from "@/pkg/hono/app";
import { newId } from "@/pkg/id";
import { KeyV1 } from "@/pkg/keys/v1";
import { testEnv } from "@/pkg/testutil/env";
import { fetchRoute } from "@/pkg/testutil/request";
import { seed } from "@/pkg/testutil/seed";
import { schema } from "@unkey/db";
import { V1KeysGetKeyResponse, registerV1KeysGetKey } from "./v1_keys_getKey";

test("returns 200", async () => {
  const env = testEnv();
  // @ts-ignore
  init({ env });
  const app = newApp();
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

  const res = await fetchRoute<never, V1KeysGetKeyResponse>(app, {
    method: "GET",
    url: `/v1/keys.getKey?keyId=${key.id}`,
    headers: {
      Authorization: `Bearer ${r.rootKey}`,
    },
  });

  expect(res.status).toEqual(200);

  expect(res.body.id).toEqual(key.id);
  expect(res.body.apiId).toEqual(r.userApi.id);
  expect(res.body.workspaceId).toEqual(key.workspaceId);
  expect(res.body.name).toEqual(key.name);
  expect(res.body.start).toEqual(key.start);
  expect(res.body.createdAt).toEqual(key.createdAt.getTime());
});
