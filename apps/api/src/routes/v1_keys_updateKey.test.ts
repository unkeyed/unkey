import { test, expect } from "bun:test";

import { init } from "@/pkg/global";
import { testEnv } from "@/pkg/testutil/env";
import { seed } from "@/pkg/testutil/seed";
import { newId } from "@/pkg/id";
import { sha256 } from "@/pkg/hash/sha256";
import { KeyV1 } from "@/pkg/keys/v1";
import { V1KeysUpdateKeyResponse, registerV1KeysUpdate } from "./v1_keys_updateKey";
import { schema } from "@unkey/db";
import { fetchRoute } from "@/pkg/testutil/request";
import { newApp } from "@/pkg/hono/app";

test("returns 200", async () => {
  const env = testEnv();
  // @ts-ignore
  init({ env });
  const app = newApp();
  registerV1KeysUpdate(app);

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

  const res = await fetchRoute<Record<string, any>, V1KeysUpdateKeyResponse>(app, {
    method: "GET",
    url: "/v1/keys.updateKey",
    headers: {
      Authorization: `Bearer ${r.rootKey}`,
    },
    body: {
      keyId: key.id,
      name: "test2",
      ownerId: "test2",
      meta: { test: "test" },
      expires: new Date(),
    },
  });

  expect(res.status).toEqual(200);
});
