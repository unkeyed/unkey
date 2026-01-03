import { randomUUID } from "node:crypto";
import { runCommonRouteTests } from "@/pkg/testutil/common-tests";
import { IntegrationHarness } from "src/pkg/testutil/integration-harness";

import { schema } from "@unkey/db";
import { newId } from "@unkey/id";
import { describe, expect, test } from "vitest";
import type {
  V1PermissionsDeleteRoleRequest,
  V1PermissionsDeleteRoleResponse,
} from "./v1_permissions_deleteRole";

runCommonRouteTests<V1PermissionsDeleteRoleRequest>({
  prepareRequest: async (rh) => {
    const roleId = newId("test");
    await rh.db.primary.insert(schema.permissions).values({
      id: roleId,
      name: randomUUID(),
      slug: randomUUID(),
      workspaceId: rh.resources.userWorkspace.id,
    });
    return {
      method: "POST",
      url: "/v1/permissions.deleteRole",
      headers: {
        "Content-Type": "application/json",
      },
      body: {
        roleId,
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

      const roleId = newId("test");
      await h.db.primary.insert(schema.roles).values({
        id: roleId,
        name: randomUUID(),
        workspaceId: h.resources.userWorkspace.id,
      });

      const root = await h.createRootKey(roles);

      const res = await h.post<V1PermissionsDeleteRoleRequest, V1PermissionsDeleteRoleResponse>({
        url: "/v1/permissions.deleteRole",
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${root.key}`,
        },
        body: {
          roleId,
        },
      });

      expect(res.status, `expected status 200, received: ${JSON.stringify(res, null, 2)}`).toEqual(
        200,
      );

      const found = await h.db.primary.query.roles.findFirst({
        where: (table, { eq }) => eq(table.id, roleId),
      });
      expect(found).toBeUndefined();
    });
  });
});
