import { randomUUID } from "node:crypto";
import { runCommonRouteTests } from "@/pkg/testutil/common-tests";
import { schema } from "@unkey/db";
import { sha256 } from "@unkey/hash";
import { newId } from "@unkey/id";
import { KeyV1 } from "@unkey/keys";
import { RouteHarness } from "src/pkg/testutil/route-harness";
import { describe, expect, test } from "vitest";
import type { V1KeysUpdateKeyRequest, V1KeysUpdateKeyResponse } from "./v1_keys_updateKey";

runCommonRouteTests<V1KeysUpdateKeyRequest>({
  prepareRequest: async (rh) => {
    const keyId = newId("key");
    const key = new KeyV1({ prefix: "test", byteLength: 16 }).toString();
    await rh.db.insert(schema.keys).values({
      id: keyId,
      keyAuthId: rh.resources.userKeyAuth.id,
      hash: await sha256(key),
      start: key.slice(0, 8),
      workspaceId: rh.resources.userWorkspace.id,
      createdAt: new Date(),
    });
    return {
      method: "POST",
      url: "/v1/keys.updateKey",
      headers: {
        "Content-Type": "application/json",
      },
      body: {
        keyId,
        enabled: false,
      },
    };
  },
});

describe("correct roles", () => {
  describe.each([
    { name: "legacy", roles: ["*"] },
    { name: "legacy and more", roles: ["*", randomUUID()] },
    { name: "wildcard api", roles: ["api.*.update_key"] },

    { name: "wildcard and more", roles: ["api.*.update_key", randomUUID()] },
    {
      name: "specific apiId",
      roles: [(apiId: string) => `api.${apiId}.update_key`],
    },
    {
      name: "specific apiId and more",
      roles: [(apiId: string) => `api.${apiId}.update_key`, randomUUID()],
    },
  ])("$name", ({ roles }) => {
    test("returns 200", async (t) => {
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

      const root = await h.createRootKey(
        roles.map((role) => (typeof role === "string" ? role : role(h.resources.userApi.id))),
      );

      const res = await h.post<V1KeysUpdateKeyRequest, V1KeysUpdateKeyResponse>({
        url: "/v1/keys.updateKey",
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${root.key}`,
        },
        body: {
          keyId,
          enabled: false,
        },
      });
      expect(res.status).toEqual(200);
    });
  });
});
