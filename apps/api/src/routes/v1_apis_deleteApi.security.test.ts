import { randomUUID } from "node:crypto";
import { runCommonRouteTests } from "@/pkg/testutil/common-tests";
import { schema } from "@unkey/db";
import { newId } from "@unkey/id";
import { RouteHarness } from "src/pkg/testutil/route-harness";
import { describe, expect, test } from "vitest";
import type { V1ApisDeleteApiRequest, V1ApisDeleteApiResponse } from "./v1_apis_deleteApi";

runCommonRouteTests<V1ApisDeleteApiRequest>({
  prepareRequest: async (rh) => {
    const apiId = newId("api");
    await rh.db.insert(schema.apis).values({
      id: apiId,
      name: randomUUID(),
      workspaceId: rh.resources.userWorkspace.id,
    });
    return {
      method: "POST",
      url: "/v1/apis.deleteApi",
      headers: {
        "Content-Type": "application/json",
      },
      body: {
        apiId,
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
      const h = await RouteHarness.init(t);
      const apiId = newId("api");
      await h.db.insert(schema.apis).values({
        id: apiId,
        name: randomUUID(),
        workspaceId: h.resources.userWorkspace.id,
      });
      const root = await h.createRootKey(
        roles.map((role) => (typeof role === "string" ? role : role(apiId))),
      );

      const res = await h.post<V1ApisDeleteApiRequest, V1ApisDeleteApiResponse>({
        url: "/v1/apis.deleteApi",
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${root.key}`,
        },
        body: {
          apiId,
        },
      });
      expect(res.status).toEqual(200);

      const found = await h.db.query.apis.findFirst({
        where: (table, { eq }) => eq(table.id, apiId),
      });
      expect(found).toBeDefined();
    });
  });
});
