import { expect, test } from "vitest";

import { randomUUID } from "node:crypto";
import { IntegrationHarness } from "src/pkg/testutil/integration-harness";

import { schema } from "@unkey/db";
import { newId } from "@unkey/id";
import type {
  V1PermissionsDeletePermissionRequest,
  V1PermissionsDeletePermissionResponse,
} from "./v1_permissions_deletePermission";

test("deletes permission", async (t) => {
  const h = await IntegrationHarness.init(t);

  const permissionId = newId("test");
  await h.db.primary.insert(schema.permissions).values({
    id: permissionId,
    name: randomUUID(),
    slug: randomUUID(),
    workspaceId: h.resources.userWorkspace.id,
  });

  const root = await h.createRootKey(["rbac.*.delete_permission"]);
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

  expect(res.status, `expected 200, received: ${JSON.stringify(res, null, 2)}`).toBe(200);

  const found = await h.db.primary.query.permissions.findFirst({
    where: (table, { eq }) => eq(table.id, permissionId),
  });
  expect(found).toBeUndefined();
});
