import { randomUUID } from "node:crypto";
import { runCommonRouteTests } from "@/pkg/testutil/common-tests";
import { IntegrationHarness } from "src/pkg/testutil/integration-harness";

import { describe, expect, test } from "vitest";
import type {
  V1PermissionsCreateRoleRequest,
  V1PermissionsCreateRoleResponse,
} from "./v1_permissions_createRole";

runCommonRouteTests<V1PermissionsCreateRoleRequest>({
  prepareRequest: () => ({
    method: "POST",
    url: "/v1/permissions.createRole",
    headers: {
      "Content-Type": "application/json",
    },
    body: {
      name: randomUUID(),
    },
  }),
});
describe("correct roles", () => {
  describe.each([
    { name: "legacy", roles: ["*"] },
    { name: "legacy and more", roles: ["*", randomUUID()] },
  ])("$name", ({ roles }) => {
    test("returns 200", async (t) => {
      const h = await IntegrationHarness.init(t);
      const root = await h.createRootKey(roles);

      const res = await h.post<V1PermissionsCreateRoleRequest, V1PermissionsCreateRoleResponse>({
        url: "/v1/permissions.createRole",
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${root.key}`,
        },
        body: {
          name: randomUUID(),
        },
      });

      expect(res.status, `expected status 200, received: ${JSON.stringify(res, null, 2)}`).toEqual(
        200,
      );

      const found = await h.db.primary.query.roles.findFirst({
        where: (table, { eq }) => eq(table.id, res.body.roleId),
      });
      expect(found).toBeDefined();
    });
  });
});
