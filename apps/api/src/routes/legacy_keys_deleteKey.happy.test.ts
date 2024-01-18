import { expect, test } from "vitest";

import { schema } from "@unkey/db";
import { sha256 } from "@unkey/hash";
import { newId } from "@unkey/id";
import { KeyV1 } from "@unkey/keys";

import { Harness } from "@/pkg/testutil/harness";
import { LegacyKeysDeleteKeyResponse, registerLegacyKeysDelete } from "./legacy_keys_deleteKey";

test("soft deletes key", async () => {
  const h = await Harness.init();
  h.useRoutes(registerLegacyKeysDelete);

  const keyId = newId("key");
  const key = new KeyV1({ prefix: "test", byteLength: 16 }).toString();
  await h.resources.database.insert(schema.keys).values({
    id: keyId,
    keyAuthId: h.resources.userKeyAuth.id,
    hash: await sha256(key),
    start: key.slice(0, 8),
    workspaceId: h.resources.userWorkspace.id,
    createdAt: new Date(),
  });

  const root = await h.createRootKey([`api.${h.resources.userApi.id}.delete_key`]);
  const res = await h.delete<LegacyKeysDeleteKeyResponse>({
    url: `/v1/keys/${keyId}`,
    headers: {
      Authorization: `Bearer ${root.key}`,
    },
  });

  expect(res.status).toEqual(200);

  const found = await h.resources.database.query.keys.findFirst({
    where: (table, { eq }) => eq(table.id, keyId),
  });
  expect(found).toBeDefined();
  expect(found!.deletedAt).toBeDefined();
  expect(found!.deletedAt!.getTime() - Date.now()).toBeLessThan(10_000); // 10s play
});
