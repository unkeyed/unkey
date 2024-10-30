import { expect, test } from "vitest";

import { randomUUID } from "node:crypto";
import { IntegrationHarness } from "src/pkg/testutil/integration-harness";

import { schema } from "@unkey/db";
import { newId } from "@unkey/id";
import type { V1KeysSetRolesRequest, V1KeysSetRolesResponse } from "./v1_keys_setRoles";

test("creates all missing roles", async (t) => {
  const h = await IntegrationHarness.init(t);
  const root = await h.createRootKey(["rbac.*.create_role", "rbac.*.add_role_to_key"]);

  const { keyId } = await h.createKey();

  const name = randomUUID();

  const res = await h.post<V1KeysSetRolesRequest, V1KeysSetRolesResponse>({
    url: "/v1/keys.setRoles",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${root.key}`,
    },
    body: {
      keyId,
      roles: [
        {
          name,
          create: true,
        },
      ],
    },
  });

  expect(res.status, `expected 200, received: ${JSON.stringify(res, null, 2)}`).toBe(200);

  const found = await h.db.primary.query.roles.findFirst({
    where: (table, { and, eq }) =>
      and(eq(table.workspaceId, h.resources.userWorkspace.id), eq(table.name, name)),
  });
  expect(found).toBeDefined();
});

test("connects all roles", async (t) => {
  const h = await IntegrationHarness.init(t);
  const root = await h.createRootKey(["rbac.*.create_permission", "rbac.*.add_role_to_key"]);

  const roles = new Array(3).fill(null).map((_) => ({
    id: newId("test"),
    name: randomUUID(),
    workspaceId: h.resources.userWorkspace.id,
  }));

  await h.db.primary.insert(schema.roles).values(roles);

  const { keyId } = await h.createKey();

  const res = await h.post<V1KeysSetRolesRequest, V1KeysSetRolesResponse>({
    url: "/v1/keys.setRoles",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${root.key}`,
    },
    body: {
      keyId,
      roles: roles.map((p) => ({ id: p.id })),
    },
  });

  expect(res.status, `expected 200, received: ${JSON.stringify(res, null, 2)}`).toBe(200);

  const key = await h.db.primary.query.keys.findFirst({
    where: (table, { eq }) => eq(table.id, keyId),
    with: {
      roles: {
        with: {
          role: true,
        },
      },
    },
  });
  expect(key).toBeDefined();
  expect(key!.roles.length).toBe(roles.length);
  for (const role of roles) {
    expect(key!.roles.some((r) => r.role.name === role.name));
  }
});

test("not desired roles are disconnected", async (t) => {
  const h = await IntegrationHarness.init(t);
  const root = await h.createRootKey([
    "rbac.*.create_role",
    "rbac.*.remove_role_from_key",
    "rbac.*.add_role_to_key",
  ]);

  const roles = new Array(3).fill(null).map((_) => ({
    id: newId("test"),
    name: randomUUID(),
    workspaceId: h.resources.userWorkspace.id,
  }));

  await h.db.primary.insert(schema.roles).values(roles);

  const { keyId } = await h.createKey();

  await h.db.primary.insert(schema.keysRoles).values({
    keyId,
    roleId: roles[0].id,
    workspaceId: h.resources.userWorkspace.id,
  });
  const res = await h.post<V1KeysSetRolesRequest, V1KeysSetRolesResponse>({
    url: "/v1/keys.setRoles",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${root.key}`,
    },
    body: {
      keyId,
      roles: roles.slice(1).map((p) => ({ id: p.id })),
    },
  });

  expect(res.status, `expected 200, received: ${JSON.stringify(res, null, 2)}`).toBe(200);

  const key = await h.db.primary.query.keys.findFirst({
    where: (table, { eq }) => eq(table.id, keyId),
    with: {
      roles: {
        with: {
          role: true,
        },
      },
    },
  });
  expect(key).toBeDefined();
  expect(key!.roles.length).toBe(2);
  for (const role of roles.slice(1)) {
    expect(key!.roles.some((r) => r.role.name === role.name));
  }
});

test("additional roles does not remove existing roles", async (t) => {
  const h = await IntegrationHarness.init(t);
  const root = await h.createRootKey(["rbac.*.create_permission", "rbac.*.add_role_to_key"]);

  const roles = new Array(3).fill(null).map((_) => ({
    id: newId("test"),
    name: randomUUID(),
    workspaceId: h.resources.userWorkspace.id,
  }));

  await h.db.primary.insert(schema.roles).values(roles);

  const { keyId } = await h.createKey();

  await h.db.primary.insert(schema.keysRoles).values({
    keyId,
    roleId: roles[0].id,
    workspaceId: h.resources.userWorkspace.id,
  });
  const res = await h.post<V1KeysSetRolesRequest, V1KeysSetRolesResponse>({
    url: "/v1/keys.setRoles",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${root.key}`,
    },
    body: {
      keyId,
      roles: roles.map((p) => ({ id: p.id })),
    },
  });

  expect(res.status, `expected 200, received: ${JSON.stringify(res, null, 2)}`).toBe(200);

  const key = await h.db.primary.query.keys.findFirst({
    where: (table, { eq }) => eq(table.id, keyId),
    with: {
      roles: {
        with: {
          role: true,
        },
      },
    },
  });
  expect(key).toBeDefined();
  expect(key!.roles.length).toBe(roles.length);
  for (const role of roles) {
    expect(key!.roles.some((r) => r.role.name === role.name));
  }
});
