import { expect, test } from "vitest";

import { schema } from "@unkey/db";
import { newId } from "@unkey/id";
import { IntegrationHarness } from "src/pkg/testutil/integration-harness";

import { randomUUID } from "node:crypto";
import type { V1PermissionsListRolesResponse } from "./v1_permissions_listRoles";

test("return all roles", async (t) => {
  const h = await IntegrationHarness.init(t);
  const root = await h.createRootKey(["rbac.*.read_role"]);

  const role = {
    id: newId("test"),
    name: randomUUID(),
    workspaceId: h.resources.userWorkspace.id,
  };
  await h.db.primary.insert(schema.roles).values(role);

  const res = await h.get<V1PermissionsListRolesResponse>({
    url: "/v1/permissions.listRoles",
    headers: {
      Authorization: `Bearer ${root.key}`,
    },
  });

  expect(res.status, `expected 200, received: ${JSON.stringify(res, null, 2)}`).toBe(200);
  expect(res.body.length).toBe(1);
  expect(res.body[0].id).toEqual(role.id);
  expect(res.body[0].name).toEqual(role.name);
  expect(res.body[0].description).toBeUndefined();
});
