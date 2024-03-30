import { randomUUID } from "node:crypto";
import { schema } from "@unkey/db";
import { sha256 } from "@unkey/hash";
import { newId } from "@unkey/id";
import { KeyV1 } from "@unkey/keys";
import { describe, expect, test } from "vitest";
import { runCommonRouteTests } from "../pkg/testutil/common-tests";
import { RouteHarness } from "../pkg/testutil/route-harness";
import type {
  V1KeysUpdateRemainingRequest,
  V1KeysUpdateRemainingResponse,
} from "./v1_keys_updateRemaining";

runCommonRouteTests<V1KeysUpdateRemainingRequest>({
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
      url: "/v1/keys.updateRemaining",
      headers: {
        "Content-Type": "application/json",
      },
      body: {
        keyId,
        op: "set",
        value: 10,
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

      const res = await h.post<V1KeysUpdateRemainingRequest, V1KeysUpdateRemainingResponse>({
        url: "/v1/keys.updateRemaining",
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${root.key}`,
        },
        body: {
          keyId,
          op: "set",
          value: 10,
        },
      });
      expect(res.status).toEqual(200);
    });
  });
});
