import { expect, test } from "vitest";

import { randomUUID } from "node:crypto";
import { IntegrationHarness } from "src/pkg/testutil/integration-harness";

import { schema } from "@unkey/db";
import { newId } from "@unkey/id";
import type {
  V1KeysSetPermissionsRequest,
  V1KeysSetPermissionsResponse,
} from "./v1_keys_setPermissions";

test("creates all missing permissions", async (t) => {
  const h = await IntegrationHarness.init(t);
  const root = await h.createRootKey(["rbac.*.create_permission", "rbac.*.add_permission_to_key"]);

  const { keyId } = await h.createKey();

  const name = randomUUID();

  const res = await h.post<V1KeysSetPermissionsRequest, V1KeysSetPermissionsResponse>({
    url: "/v1/keys.setPermissions",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${root.key}`,
    },
    body: {
      keyId,
      permissions: [
        {
          name,
          create: true,
        },
      ],
    },
  });

  expect(res.status, `expected 200, received: ${JSON.stringify(res, null, 2)}`).toBe(200);

  const found = await h.db.primary.query.permissions.findFirst({
    where: (table, { and, eq }) =>
      and(eq(table.workspaceId, h.resources.userWorkspace.id), eq(table.name, name)),
  });
  expect(found).toBeDefined();
});

test("connects all permissions", async (t) => {
  const h = await IntegrationHarness.init(t);
  const root = await h.createRootKey(["rbac.*.create_permission", "rbac.*.add_permission_to_key"]);

  const permissions = new Array(3).fill(null).map((_) => ({
    id: newId("test"),
    name: randomUUID(),
    slug: randomUUID(),
    workspaceId: h.resources.userWorkspace.id,
  }));

  await h.db.primary.insert(schema.permissions).values(permissions);

  const { keyId } = await h.createKey();

  const res = await h.post<V1KeysSetPermissionsRequest, V1KeysSetPermissionsResponse>({
    url: "/v1/keys.setPermissions",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${root.key}`,
    },
    body: {
      keyId,
      permissions: permissions.map((p) => ({ id: p.id })),
    },
  });

  expect(res.status, `expected 200, received: ${JSON.stringify(res, null, 2)}`).toBe(200);

  const key = await h.db.primary.query.keys.findFirst({
    where: (table, { eq }) => eq(table.id, keyId),
    with: {
      permissions: {
        with: {
          permission: true,
        },
      },
    },
  });
  expect(key).toBeDefined();
  expect(key!.permissions.length).toBe(permissions.length);
  for (const permission of permissions) {
    expect(key!.permissions.some((r) => r.permission.name === permission.name));
  }
});

test("not desired permissions are removed", async (t) => {
  const h = await IntegrationHarness.init(t);
  const root = await h.createRootKey([
    "rbac.*.create_permission",
    "rbac.*.remove_permission_from_key",
    "rbac.*.add_permission_to_key",
  ]);

  const permissions = new Array(3).fill(null).map((_) => ({
    id: newId("test"),
    name: randomUUID(),
    slug: randomUUID(),
    workspaceId: h.resources.userWorkspace.id,
  }));

  await h.db.primary.insert(schema.permissions).values(permissions);

  const { keyId } = await h.createKey();

  await h.db.primary.insert(schema.keysPermissions).values({
    keyId,
    permissionId: permissions[0].id,
    workspaceId: h.resources.userWorkspace.id,
  });
  const res = await h.post<V1KeysSetPermissionsRequest, V1KeysSetPermissionsResponse>({
    url: "/v1/keys.setPermissions",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${root.key}`,
    },
    body: {
      keyId,
      permissions: permissions.slice(1).map((p) => ({ id: p.id })),
    },
  });

  expect(res.status, `expected 200, received: ${JSON.stringify(res, null, 2)}`).toBe(200);

  const key = await h.db.primary.query.keys.findFirst({
    where: (table, { eq }) => eq(table.id, keyId),
    with: {
      permissions: {
        with: {
          permission: true,
        },
      },
    },
  });
  expect(key).toBeDefined();
  expect(key!.permissions.length).toBe(2);
  for (const permission of permissions.slice(1)) {
    expect(key!.permissions.some((r) => r.permission.name === permission.name));
  }
});

test("additional permissions does not remove existing permissions", async (t) => {
  const h = await IntegrationHarness.init(t);
  const root = await h.createRootKey(["rbac.*.create_permission", "rbac.*.add_permission_to_key"]);

  const permissions = new Array(3).fill(null).map((_) => ({
    id: newId("test"),
    name: randomUUID(),
    slug: randomUUID(),
    workspaceId: h.resources.userWorkspace.id,
  }));

  await h.db.primary.insert(schema.permissions).values(permissions);

  const { keyId } = await h.createKey();

  await h.db.primary.insert(schema.keysPermissions).values({
    keyId,
    permissionId: permissions[0].id,
    workspaceId: h.resources.userWorkspace.id,
  });
  const res = await h.post<V1KeysSetPermissionsRequest, V1KeysSetPermissionsResponse>({
    url: "/v1/keys.setPermissions",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${root.key}`,
    },
    body: {
      keyId,
      permissions: permissions.map((p) => ({ id: p.id })),
    },
  });

  expect(res.status, `expected 200, received: ${JSON.stringify(res, null, 2)}`).toBe(200);

  const key = await h.db.primary.query.keys.findFirst({
    where: (table, { eq }) => eq(table.id, keyId),
    with: {
      permissions: {
        with: {
          permission: true,
        },
      },
    },
  });
  expect(key).toBeDefined();
  expect(key!.permissions.length).toBe(permissions.length);
  for (const permission of permissions) {
    expect(key!.permissions.some((r) => r.permission.name === permission.name));
  }
});
