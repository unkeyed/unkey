import { randomUUID } from "node:crypto";
import { runCommonRouteTests } from "@/pkg/testutil/common-tests";
import { schema } from "@unkey/db";
import { newId } from "@unkey/id";
import { IntegrationHarness } from "src/pkg/testutil/integration-harness";

import { describe, expect, test } from "vitest";
import type { V1ApisDeleteKeysRequest, V1ApisDeleteKeysResponse } from "./v1_apis_deleteKeys";

runCommonRouteTests<V1ApisDeleteKeysRequest>({
  prepareRequest: async (rh) => {
    const apiId = newId("test");
    await rh.db.primary.insert(schema.apis).values({
      id: apiId,
      name: randomUUID(),
      workspaceId: rh.resources.userWorkspace.id,
    });
    return {
      method: "POST",
      url: "/v1/apis.deleteKeys",
      headers: {
        "Content-Type": "application/json",
      },
      body: {
        apiId,
        softDelete: true,
      },
    };
  },
});

describe("correct permissions", () => {
  describe.each([
    { name: "legacy", permissions: ["*"] },
    { name: "legacy and more", permissions: ["*", randomUUID()] },
    { name: "wildcard", permissions: ["api.*.delete_key"] },
    { name: "wildcard and more", permissions: ["api.*.delete_key", randomUUID()] },
    { name: "specific apiId", permissions: [(apiId: string) => `api.${apiId}.delete_key`] },
    {
      name: "specific apiId and more",
      permissions: [(apiId: string) => `api.${apiId}.delete_key`, randomUUID()],
    },
  ])("$name", ({ permissions }) => {
    test("returns 200", async (t) => {
      const h = await IntegrationHarness.init(t);
      const apiId = newId("test");
      const keyAuthId = newId("test");
      await h.db.primary.insert(schema.keyAuth).values({
        id: keyAuthId,
        workspaceId: h.resources.userWorkspace.id,
      });
      await h.db.primary.insert(schema.apis).values({
        id: apiId,
        keyAuthId,
        name: randomUUID(),
        workspaceId: h.resources.userWorkspace.id,
      });
      const root = await h.createRootKey(
        permissions.map((permission) =>
          typeof permission === "string" ? permission : permission(apiId),
        ),
      );

      for (let i = 0; i < 10; i++) {
        await h.createKey();
      }

      const res = await h.post<V1ApisDeleteKeysRequest, V1ApisDeleteKeysResponse>({
        url: "/v1/apis.deleteKeys",
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${root.key}`,
        },
        body: {
          apiId,
        },
      });
      expect(res.status, `expected 200, received: ${JSON.stringify(res, null, 2)}`).toBe(200);

      const found = await h.db.primary.query.apis.findFirst({
        where: (table, { eq }) => eq(table.id, apiId),
        with: {
          keyAuth: {
            with: {
              keys: true,
            },
          },
        },
      });
      expect(found).toBeDefined();
      expect(found!.keyAuth!.keys.length).toEqual(0);
    });
  });
});
