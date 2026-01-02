import { randomUUID } from "node:crypto";
import { runCommonRouteTests } from "@/pkg/testutil/common-tests";
import { schema } from "@unkey/db";
import { newId } from "@unkey/id";
import { IntegrationHarness } from "src/pkg/testutil/integration-harness";

import { describe, expect, test } from "vitest";
import type { V1PermissionsListPermissionsResponse } from "./v1_permissions_listPermissions";

runCommonRouteTests({
  prepareRequest: async (rh) => {
    const permissionId = newId("test");
    await rh.db.primary.insert(schema.permissions).values({
      id: permissionId,
      name: randomUUID(),
      slug: randomUUID(),
      workspaceId: rh.resources.userWorkspace.id,
    });
    return {
      method: "GET",
      url: "/v1/permissions.listPermissions",
    };
  },
});

describe("correct permissions", () => {
  describe.each([
    { name: "legacy", permissions: ["*"] },
    { name: "legacy and more", permissions: ["*", randomUUID()] },
    { name: "wildcard", permissions: ["rbac.*.read_permission"] },
    { name: "wildcard and more", permissions: ["rbac.*.read_permission", randomUUID()] },
  ])("$name", ({ permissions }) => {
    test("returns 200", async (t) => {
      const h = await IntegrationHarness.init(t);
      const permissionId = newId("test");
      await h.db.primary.insert(schema.permissions).values({
        id: permissionId,
        name: randomUUID(),
        slug: randomUUID(),
        workspaceId: h.resources.userWorkspace.id,
      });
      const root = await h.createRootKey(permissions);

      const res = await h.get<V1PermissionsListPermissionsResponse>({
        url: "/v1/permissions.listPermissions",
        headers: {
          Authorization: `Bearer ${root.key}`,
        },
      });
      expect(res.status, `expected 200, received: ${JSON.stringify(res, null, 2)}`).toBe(200);
    });
  });
});
