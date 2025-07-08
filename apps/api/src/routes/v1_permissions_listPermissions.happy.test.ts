import { expect, test } from "vitest";

import { schema } from "@unkey/db";
import { newId } from "@unkey/id";
import { IntegrationHarness } from "src/pkg/testutil/integration-harness";

import { randomUUID } from "node:crypto";
import type { V1PermissionsListPermissionsResponse } from "./v1_permissions_listPermissions";

test("return all permissions", async (t) => {
  const h = await IntegrationHarness.init(t);
  const root = await h.createRootKey(["rbac.*.read_permission"]);

  const permission = {
    id: newId("test"),
    name: randomUUID(),
    slug: randomUUID(),
    workspaceId: h.resources.userWorkspace.id,
  };
  await h.db.primary.insert(schema.permissions).values(permission);

  const res = await h.get<V1PermissionsListPermissionsResponse>({
    url: "/v1/permissions.listPermissions",
    headers: {
      Authorization: `Bearer ${root.key}`,
    },
  });

  expect(res.status, `expected 200, received: ${JSON.stringify(res, null, 2)}`).toBe(200);
  expect(res.body.length).toBe(1);
  expect(res.body[0].id).toEqual(permission.id);
  expect(res.body[0].name).toEqual(permission.name);
  expect(res.body[0].slug).toEqual(permission.slug);
  expect(res.body[0].description).toBeUndefined();
});
