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

import { DeleteKeyResponse, registerDeleteKey } from "./keys_delete";

test("soft deletes key", async () => {
  const env = testEnv();
  // @ts-ignore
  init({ env });
  const app = newApp();
  registerDeleteKey(app);

  const r = await seed(env);

  const keyId = newId("key");
  const key = new KeyV1({ prefix: "test", byteLength: 16 }).toString();
  await r.database.insert(schema.keys).values({
    id: keyId,
    keyAuthId: r.userKeyAuth.id,
    hash: await sha256(key),
    start: key.slice(0, 8),
    workspaceId: r.userWorkspace.id,
    createdAt: new Date(),
  });

  const res = await fetchRoute<never, DeleteKeyResponse>(app, {
    method: "DELETE",
    url: `/v1/keys/${keyId}`,
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${r.rootKey}`,
    },
  });

  expect(res.status).toEqual(200);

  const found = await r.database.query.keys.findFirst({
    where: (table, { eq }) => eq(table.id, keyId),
  });
  expect(found).toBeDefined();
  expect(found!.deletedAt).toBeDefined();
  expect(found!.deletedAt!.getTime()).toBeWithin(Date.now() - 10_000, Date.now());
});
