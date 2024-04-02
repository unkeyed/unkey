import { expect, test } from "vitest";

import { schema } from "@unkey/db";
import { sha256 } from "@unkey/hash";
import { newId } from "@unkey/id";
import { KeyV1 } from "@unkey/keys";

import { RouteHarness } from "src/pkg/testutil/route-harness";
import type { V1KeysDeleteKeyRequest, V1KeysDeleteKeyResponse } from "./v1_keys_deleteKey";

test("soft deletes key", async (t) => {
  const h = await RouteHarness.init(t);
  const keyId = newId("key");
  const key = new KeyV1({ prefix: "test", byteLength: 16 }).toString();
  await h.db.insert(schema.keys).values({
    id: keyId,
    keyAuthId: h.resources.userKeyAuth.id,
    hash: await sha256(key),
    start: key.slice(0, 8),
    workspaceId: h.resources.userWorkspace.id,
    createdAt: new Date(),
  });

  const root = await h.createRootKey([`api.${h.resources.userApi.id}.delete_key`]);
  const res = await h.post<V1KeysDeleteKeyRequest, V1KeysDeleteKeyResponse>({
    url: "/v1/keys.deleteKey",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${root.key}`,
    },
    body: {
      keyId,
    },
  });

  expect(res.status).toEqual(200);

  const found = await h.db.query.keys.findFirst({
    where: (table, { eq }) => eq(table.id, keyId),
  });
  expect(found).toBeDefined();
  expect(found!.deletedAt).toBeDefined();
  expect(found!.deletedAt!.getTime() - Date.now()).toBeLessThan(10_000); // 10s play
});
