import { expect, test } from "vitest";

import { randomUUID } from "node:crypto";
import { IntegrationHarness } from "src/pkg/testutil/integration-harness";

import { schema } from "@unkey/db";
import { newId } from "@unkey/id";
import type {
  V1PermissionsDeleteRoleRequest,
  V1PermissionsDeleteRoleResponse,
} from "./v1_permissions_deleteRole";

test("deletes role", async (t) => {
  const h = await IntegrationHarness.init(t);

  const roleId = newId("test");
  await h.db.primary.insert(schema.roles).values({
    id: roleId,
    name: randomUUID(),
    workspaceId: h.resources.userWorkspace.id,
  });

  const root = await h.createRootKey(["rbac.*.delete_role"]);
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

  expect(res.status, `expected 200, received: ${JSON.stringify(res, null, 2)}`).toBe(200);

  const found = await h.db.primary.query.roles.findFirst({
    where: (table, { eq }) => eq(table.id, roleId),
  });
  expect(found).toBeUndefined();
});
