import { randomUUID } from "node:crypto";
import { runCommonRouteTests } from "@/pkg/testutil/common-tests";
import { IntegrationHarness } from "src/pkg/testutil/integration-harness";

import { describe, expect, test } from "vitest";
import type {
  V1PermissionsCreatePermissionRequest,
  V1PermissionsCreatePermissionResponse,
} from "./v1_permissions_createPermission";

runCommonRouteTests<V1PermissionsCreatePermissionRequest>({
  prepareRequest: () => ({
    method: "POST",
    url: "/v1/permissions.createPermission",
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

      const res = await h.post<
        V1PermissionsCreatePermissionRequest,
        V1PermissionsCreatePermissionResponse
      >({
        url: "/v1/permissions.createPermission",
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

      const found = await h.db.primary.query.permissions.findFirst({
        where: (table, { eq }) => eq(table.id, res.body.permissionId),
      });
      expect(found).toBeDefined();
    });
  });
});
