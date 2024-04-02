import { randomUUID } from "node:crypto";
import { runCommonRouteTests } from "@/pkg/testutil/common-tests";
import { schema } from "@unkey/db";
import { newId } from "@unkey/id";
import { RouteHarness } from "src/pkg/testutil/route-harness";
import { describe, expect, test } from "vitest";
import type { V1KeysCreateKeyRequest, V1KeysCreateKeyResponse } from "./v1_keys_createKey";

runCommonRouteTests<V1KeysCreateKeyRequest>({
  prepareRequest: async (rh) => {
    const apiId = newId("api");
    await rh.db.insert(schema.apis).values({
      id: apiId,
      name: randomUUID(),
      workspaceId: rh.resources.userWorkspace.id,
    });
    return {
      method: "POST",
      url: "/v1/keys.createKey",
      headers: {
        "Content-Type": "application/json",
      },
      body: {
        apiId,
        byteLength: 16,
      },
    };
  },
});

describe("correct roles", () => {
  describe.each([
    { name: "legacy", roles: ["*"] },
    { name: "legacy and more", roles: ["*", randomUUID()] },
    { name: "wildcard api", roles: ["api.*.create_key"] },

    { name: "wildcard and more", roles: ["api.*.create_key", randomUUID()] },
    {
      name: "specific apiId",
      roles: [(apiId: string) => `api.${apiId}.create_key`],
    },
    {
      name: "specific apiId and more",
      roles: [(apiId: string) => `api.${apiId}.create_key`, randomUUID()],
    },
  ])("$name", ({ roles }) => {
    test("returns 200", async (t) => {
      const h = await RouteHarness.init(t);
      const keyAuthId = newId("keyAuth");
      await h.db.insert(schema.keyAuth).values({
        id: keyAuthId,
        workspaceId: h.resources.userWorkspace.id,
      });

      const apiId = newId("api");
      await h.db.insert(schema.apis).values({
        id: apiId,
        name: randomUUID(),
        workspaceId: h.resources.userWorkspace.id,
        authType: "key",
        keyAuthId,
      });

      const root = await h.createRootKey(
        roles.map((role) => (typeof role === "string" ? role : role(apiId))),
      );

      const res = await h.post<V1KeysCreateKeyRequest, V1KeysCreateKeyResponse>({
        url: "/v1/keys.createKey",
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${root.key}`,
        },
        body: {
          apiId,
        },
      });
      expect(res.status).toEqual(200);
    });
  });
});
