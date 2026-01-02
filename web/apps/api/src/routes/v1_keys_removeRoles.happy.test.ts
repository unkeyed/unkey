import { expect, test } from "vitest";

import { randomUUID } from "node:crypto";
import { IntegrationHarness } from "src/pkg/testutil/integration-harness";

import { schema } from "@unkey/db";
import { newId } from "@unkey/id";
import type { V1KeysRemoveRolesRequest, V1KeysRemoveRolesResponse } from "./v1_keys_removeRoles";

test("removes role by name", async (t) => {
  const h = await IntegrationHarness.init(t);
  const root = await h.createRootKey(["rbac.*.remove_role_from_key"]);

  const { keyId } = await h.createKey();

  const role = {
    id: newId("test"),
    name: randomUUID(),
    workspaceId: h.resources.userWorkspace.id,
  };

  await h.db.primary.insert(schema.roles).values(role);
  await h.db.primary.insert(schema.keysRoles).values({
    workspaceId: h.resources.userWorkspace.id,
    keyId,
    roleId: role.id,
  });

  const res = await h.post<V1KeysRemoveRolesRequest, V1KeysRemoveRolesResponse>({
    url: "/v1/keys.removeRoles",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${root.key}`,
    },
    body: {
      keyId,
      roles: [
        {
          name: role.name,
        },
      ],
    },
  });

  expect(res.status, `expected 200, received: ${JSON.stringify(res, null, 2)}`).toBe(200);

  const key = await h.db.primary.query.keys.findFirst({
    where: (table, { and, eq }) =>
      and(eq(table.workspaceId, h.resources.userWorkspace.id), eq(table.id, keyId)),
    with: {
      roles: true,
    },
  });
  expect(key).toBeDefined();
  expect(key!.roles.length).toBe(0);
});

test("removes role by id", async (t) => {
  const h = await IntegrationHarness.init(t);
  const root = await h.createRootKey(["rbac.*.remove_role_from_key"]);

  const { keyId } = await h.createKey();

  const role = {
    id: newId("test"),
    name: randomUUID(),
    workspaceId: h.resources.userWorkspace.id,
  };

  await h.db.primary.insert(schema.roles).values(role);
  await h.db.primary.insert(schema.keysRoles).values({
    workspaceId: h.resources.userWorkspace.id,
    keyId,
    roleId: role.id,
  });

  const res = await h.post<V1KeysRemoveRolesRequest, V1KeysRemoveRolesResponse>({
    url: "/v1/keys.removeRoles",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${root.key}`,
    },
    body: {
      keyId,
      roles: [
        {
          id: role.id,
        },
      ],
    },
  });

  expect(res.status, `expected 200, received: ${JSON.stringify(res, null, 2)}`).toBe(200);

  const key = await h.db.primary.query.keys.findFirst({
    where: (table, { and, eq }) =>
      and(eq(table.workspaceId, h.resources.userWorkspace.id), eq(table.id, keyId)),
    with: {
      roles: true,
    },
  });
  expect(key).toBeDefined();
  expect(key!.roles.length).toBe(0);
});
