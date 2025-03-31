import { expect, test } from "vitest";

import { randomUUID } from "node:crypto";
import { IntegrationHarness } from "src/pkg/testutil/integration-harness";

import type {
  V1PermissionsCreateRoleRequest,
  V1PermissionsCreateRoleResponse,
} from "./v1_permissions_createRole";

test("creates new role", async (t) => {
  const h = await IntegrationHarness.init(t);
  const root = await h.createRootKey(["rbac.*.create_role"]);
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

  expect(res.status, `expected 200, received: ${JSON.stringify(res, null, 2)}`).toBe(200);

  const found = await h.db.primary.query.roles.findFirst({
    where: (table, { eq }) => eq(table.id, res.body.roleId),
  });
  expect(found).toBeDefined();
});
