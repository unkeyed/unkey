import { schema } from "@unkey/db";
import { sha256 } from "@unkey/hash";
import { newId } from "@unkey/id";
import { KeyV1 } from "@unkey/keys";

import { randomUUID } from "crypto";
import { Harness } from "@/pkg/testutil/harness";
import { runSharedRoleTests } from "@/pkg/testutil/test_route_roles";
import { describe, expect, test } from "vitest";
import { LegacyKeysDeleteKeyResponse, registerLegacyKeysDelete } from "./legacy_keys_deleteKey";
runSharedRoleTests<LegacyKeysDeleteKeyResponse>({
  registerHandler: registerLegacyKeysDelete,
  prepareRequest: async (h) => {
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

    return {
      method: "DELETE",
      url: `/v1/keys/${keyId}`,
    };
  },
});

describe("correct roles", () => {
  test.each([
    { name: "legacy", roles: ["*"] },
    { name: "legacy and more", roles: ["*", randomUUID()] },
    { name: "wildcard", roles: ["api.*.delete_key"] },
    { name: "wildcard and more", roles: ["api.*.delete_key", randomUUID()] },
    { name: "specific apiId", roles: [(apiId: string) => `api.${apiId}.delete_key`] },
    {
      name: "specific apiId and more",
      roles: [(apiId: string) => `api.${apiId}.delete_key`, randomUUID()],
    },
  ])("$name", async ({ roles }) => {
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

    const root = await h.createRootKey(
      roles.map((role) => (typeof role === "string" ? role : role(h.resources.userApi.id))),
    );

    const res = await h.delete({
      url: `/v1/keys/${keyId}`,
      headers: {
        Authorization: `Bearer ${root.key}`,
      },
    });
    console.log(res);
    expect(res.status).toEqual(200);

    const found = await h.resources.database.query.keys.findFirst({
      where: (table, { eq }) => eq(table.id, keyId),
    });
    expect(found).toBeDefined();
    expect(found!.deletedAt).toBeDefined();
  });
});
