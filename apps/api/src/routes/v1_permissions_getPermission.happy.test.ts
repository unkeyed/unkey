import { expect, test } from "vitest";

import { schema } from "@unkey/db";
import { newId } from "@unkey/id";
import { IntegrationHarness } from "src/pkg/testutil/integration-harness";

import { randomUUID } from "node:crypto";
import type { V1PermissionsGetPermissionResponse } from "./v1_permissions_getPermission";

test("return the role", async (t) => {
  const h = await IntegrationHarness.init(t);
  const root = await h.createRootKey(["rbac.*.read_permission"]);

  const permission = {
    id: newId("test"),
    name: randomUUID(),
    workspaceId: h.resources.userWorkspace.id,
    slug: randomUUID(),
  };
  await h.db.primary.insert(schema.permissions).values(permission);

  const res = await h.get<V1PermissionsGetPermissionResponse>({
    url: `/v1/permissions.getPermission?permissionId=${permission.id}`,
    headers: {
      Authorization: `Bearer ${root.key}`,
    },
  });

  expect(res.status, `expected 200, received: ${JSON.stringify(res, null, 2)}`).toBe(200);
  expect(res.body.id).toEqual(permission.id);
  expect(res.body.name).toEqual(permission.name);
  expect(res.body.slug).toEqual(permission.slug);
  expect(res.body.description).toBeUndefined();
});
