import { randomUUID } from "node:crypto";
import { runCommonRouteTests } from "@/pkg/testutil/common-tests";
import { IntegrationHarness } from "src/pkg/testutil/integration-harness";

import { schema } from "@unkey/db";
import { newId } from "@unkey/id";
import { describe, expect, test } from "vitest";
import type {
  V1PermissionsDeletePermissionRequest,
  V1PermissionsDeletePermissionResponse,
} from "./v1_permissions_deletePermission";

runCommonRouteTests<V1PermissionsDeletePermissionRequest>({
  prepareRequest: async (rh) => {
    const permissionId = newId("test");
    await rh.db.primary.insert(schema.permissions).values({
      id: permissionId,
      name: randomUUID(),
      slug: randomUUID(),
      workspaceId: rh.resources.userWorkspace.id,
    });
    return {
      method: "POST",
      url: "/v1/permissions.deletePermission",
      headers: {
        "Content-Type": "application/json",
      },
      body: {
        permissionId,
      },
    };
  },
});
describe("correct roles", () => {
  describe.each([
    { name: "legacy", roles: ["*"] },
    { name: "legacy and more", roles: ["*", randomUUID()] },
  ])("$name", ({ roles }) => {
    test("returns 200", async (t) => {
      const h = await IntegrationHarness.init(t);

      const permissionId = newId("test");
      await h.db.primary.insert(schema.permissions).values({
        id: permissionId,
        name: randomUUID(),
        slug: randomUUID(),
        workspaceId: h.resources.userWorkspace.id,
      });

      const root = await h.createRootKey(roles);

      const res = await h.post<
        V1PermissionsDeletePermissionRequest,
        V1PermissionsDeletePermissionResponse
      >({
        url: "/v1/permissions.deletePermission",
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${root.key}`,
        },
        body: {
          permissionId,
        },
      });

      expect(res.status, `expected status 200, received: ${JSON.stringify(res, null, 2)}`).toEqual(
        200,
      );

      const found = await h.db.primary.query.permissions.findFirst({
        where: (table, { eq }) => eq(table.id, permissionId),
      });
      expect(found).toBeUndefined();
    });
  });
});
