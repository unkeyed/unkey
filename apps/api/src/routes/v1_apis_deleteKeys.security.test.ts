import { randomUUID } from "node:crypto";
import { runCommonRouteTests } from "@/pkg/testutil/common-tests";
import { schema } from "@unkey/db";
import { newId } from "@unkey/id";
import { IntegrationHarness } from "src/pkg/testutil/integration-harness";

import { describe, expect, test } from "vitest";
import type { V1ApisDeleteKeysRequest, V1ApisDeleteKeysResponse } from "./v1_apis_deleteKeys";

runCommonRouteTests<V1ApisDeleteKeysRequest>({
  prepareRequest: async (rh) => {
    const apiId = newId("api");
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

describe("correct roles", () => {
  describe.each([
    { name: "legacy", roles: ["*"] },
    { name: "legacy and more", roles: ["*", randomUUID()] },
    { name: "wildcard", roles: ["api.*.delete_api"] },
    { name: "wildcard and more", roles: ["api.*.delete_api", randomUUID()] },
    { name: "specific apiId", roles: [(apiId: string) => `api.${apiId}.delete_api`] },
    {
      name: "specific apiId and more",
      roles: [(apiId: string) => `api.${apiId}.delete_api`, randomUUID()],
    },
  ])("$name", ({ roles }) => {
    test("returns 200", async (t) => {
      const h = await IntegrationHarness.init(t);
      const apiId = newId("api");
      await h.db.primary.insert(schema.apis).values({
        id: apiId,
        name: randomUUID(),
        workspaceId: h.resources.userWorkspace.id,
      });
      const root = await h.createRootKey(
        roles.map((role) => (typeof role === "string" ? role : role(apiId))),
      );

      const res = await h.post<V1ApisDeleteKeysRequest, V1ApisDeleteKeysResponse>({
        url: "/v1/apis.deleteApi",
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${root.key}`,
        },
        body: {
          apiId,
          permanent: true,
        },
      });
      expect(res.status, `expected 200, received: ${JSON.stringify(res)}`).toBe(200);

      const found = await h.db.readonly.query.apis.findFirst({
        where: (table, { eq }) => eq(table.id, apiId),
      });
      expect(found).toBeDefined();
    });
  });
});
