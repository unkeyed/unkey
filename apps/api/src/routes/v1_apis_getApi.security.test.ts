import { randomUUID } from "node:crypto";
import { runCommonRouteTests } from "@/pkg/testutil/common-tests";
import { schema } from "@unkey/db";
import { newId } from "@unkey/id";
import { IntegrationHarness } from "src/pkg/testutil/integration-harness";

import { describe, expect, test } from "vitest";
import type { V1ApisGetApiResponse } from "./v1_apis_getApi";

runCommonRouteTests({
  prepareRequest: async (rh) => {
    const apiId = newId("api");
    await rh.db.primary.insert(schema.apis).values({
      id: apiId,
      name: randomUUID(),
      workspaceId: rh.resources.userWorkspace.id,
    });
    return {
      method: "GET",
      url: `/v1/apis.getApi?apiId=${apiId}`,
    };
  },
});

describe("correct roles", () => {
  describe.each([
    { name: "legacy", roles: ["*"] },
    { name: "legacy and more", roles: ["*", randomUUID()] },
    { name: "wildcard", roles: ["api.*.read_api"] },
    { name: "wildcard and more", roles: ["api.*.read_api", randomUUID()] },
    { name: "specific apiId", roles: [(apiId: string) => `api.${apiId}.read_api`] },
    {
      name: "specific apiId and more",
      roles: [(apiId: string) => `api.${apiId}.read_api`, randomUUID()],
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

      const res = await h.get<V1ApisGetApiResponse>({
        url: `/v1/apis.getApi?apiId=${apiId}`,
        headers: {
          Authorization: `Bearer ${root.key}`,
        },
      });
      expect(res.status, `expected 200, received: ${JSON.stringify(res, null, 2)}`).toBe(200);
    });
  });
});
