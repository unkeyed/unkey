import { describe, expect, test } from "vitest";

import { schema } from "@unkey/db";
import { sha256 } from "@unkey/hash";
import { newId } from "@unkey/id";
import { KeyV1 } from "@unkey/keys";
import { IntegrationHarness } from "src/pkg/testutil/integration-harness";

import { randomUUID } from "node:crypto";
import type { V1KeysCreateKeyRequest, V1KeysCreateKeyResponse } from "./v1_keys_createKey";
import type { V1KeysUpdateKeyRequest, V1KeysUpdateKeyResponse } from "./v1_keys_updateKey";
import type { V1KeysVerifyKeyRequest, V1KeysVerifyKeyResponse } from "./v1_keys_verifyKey";

test("returns 200", async (t) => {
  const h = await IntegrationHarness.init(t);

  const key = {
    id: newId("test"),
    keyAuthId: h.resources.userKeyAuth.id,
    workspaceId: h.resources.userWorkspace.id,
    start: "test",
    name: "test",
    hash: await sha256(new KeyV1({ byteLength: 16 }).toString()),
    createdAtM: Date.now(),
  };
  await h.db.primary.insert(schema.keys).values(key);

  const root = await h.createRootKey([`api.${h.resources.userApi.id}.update_key`]);

  const res = await h.post<V1KeysUpdateKeyRequest, V1KeysUpdateKeyResponse>({
    url: "/v1/keys.updateKey",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${root.key}`,
    },
    body: {
      keyId: key.id,
      name: "test2",
      ownerId: "test2",
      meta: { test: "test" },
      expires: Date.now(),
      enabled: true,
    },
  });

  expect(res.status, `expected 200, received: ${JSON.stringify(res, null, 2)}`).toBe(200);
});

test("update all", async (t) => {
  const h = await IntegrationHarness.init(t);

  const key = {
    id: newId("test"),
    keyAuthId: h.resources.userKeyAuth.id,
    workspaceId: h.resources.userWorkspace.id,
    start: "test",
    name: "test",
    hash: await sha256(new KeyV1({ byteLength: 16 }).toString()),
    createdAtM: Date.now(),
  };
  await h.db.primary.insert(schema.keys).values(key);
  const root = await h.createRootKey([`api.${h.resources.userApi.id}.update_key`]);

  const res = await h.post<V1KeysUpdateKeyRequest, V1KeysUpdateKeyResponse>({
    url: "/v1/keys.updateKey",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${root.key}`,
    },
    body: {
      keyId: key.id,
      name: "newName",
      ownerId: "newOwnerId",
      expires: null,
      meta: { new: "meta" },
      ratelimit: {
        type: "fast",
        limit: 10,
        refillRate: 5,
        refillInterval: 1000,
      },
      remaining: 0,
      enabled: true,
    },
  });

  expect(res.status, `expected 200, received: ${JSON.stringify(res, null, 2)}`).toBe(200);

  const found = await h.db.primary.query.keys.findFirst({
    where: (table, { eq }) => eq(table.id, key.id),
    with: {
      ratelimits: true,
      credits: true,
    },
  });
  expect(found).toBeDefined();
  expect(found?.name).toEqual("newName");
  expect(found?.ownerId).toEqual("newOwnerId");
  expect(found?.meta).toEqual(JSON.stringify({ new: "meta" }));
  // Since key didn't have legacy remaining, the new credits table is used
  expect(found?.remaining).toBeNull();
  expect(found?.credits).toBeDefined();
  expect(found?.credits?.remaining).toEqual(0);
  expect(found!.ratelimits.length).toBe(1);
  expect(found!.ratelimits[0].name).toBe("default");
  expect(found!.ratelimits[0].limit).toBe(10);
  expect(found!.ratelimits[0].duration).toBe(1000);
});

test("update all with legacy remaining", async (t) => {
  const h = await IntegrationHarness.init(t);

  const key = {
    id: newId("test"),
    keyAuthId: h.resources.userKeyAuth.id,
    workspaceId: h.resources.userWorkspace.id,
    start: "test",
    name: "test",
    hash: await sha256(new KeyV1({ byteLength: 16 }).toString()),
    remaining: 100,
    createdAtM: Date.now(),
  };
  await h.db.primary.insert(schema.keys).values(key);
  const root = await h.createRootKey([`api.${h.resources.userApi.id}.update_key`]);

  const res = await h.post<V1KeysUpdateKeyRequest, V1KeysUpdateKeyResponse>({
    url: "/v1/keys.updateKey",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${root.key}`,
    },
    body: {
      keyId: key.id,
      name: "newName",
      ownerId: "newOwnerId",
      expires: null,
      meta: { new: "meta" },
      ratelimit: {
        type: "fast",
        limit: 10,
        refillRate: 5,
        refillInterval: 1000,
      },
      remaining: 50,
      enabled: true,
    },
  });

  expect(res.status, `expected 200, received: ${JSON.stringify(res, null, 2)}`).toBe(200);

  const found = await h.db.primary.query.keys.findFirst({
    where: (table, { eq }) => eq(table.id, key.id),
    with: {
      ratelimits: true,
      credits: true,
    },
  });
  expect(found).toBeDefined();
  expect(found?.name).toEqual("newName");
  expect(found?.ownerId).toEqual("newOwnerId");
  expect(found?.meta).toEqual(JSON.stringify({ new: "meta" }));
  // Since key has legacy remaining, it should stay in legacy system
  expect(found?.remaining).toEqual(50);
  expect(found?.credits).toBeNull();
  expect(found!.ratelimits.length).toBe(1);
  expect(found!.ratelimits[0].name).toBe("default");
  expect(found!.ratelimits[0].limit).toBe(10);
  expect(found!.ratelimits[0].duration).toBe(1000);
});

test("update ratelimit", async (t) => {
  const h = await IntegrationHarness.init(t);

  const key = {
    id: newId("test"),
    keyAuthId: h.resources.userKeyAuth.id,
    workspaceId: h.resources.userWorkspace.id,
    start: "test",
    name: "test",
    hash: await sha256(new KeyV1({ byteLength: 16 }).toString()),
    createdAtM: Date.now(),
  };
  await h.db.primary.insert(schema.keys).values(key);
  const root = await h.createRootKey([`api.${h.resources.userApi.id}.update_key`]);

  const res = await h.post<V1KeysUpdateKeyRequest, V1KeysUpdateKeyResponse>({
    url: "/v1/keys.updateKey",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${root.key}`,
    },
    body: {
      keyId: key.id,
      ratelimit: {
        type: "fast",
        limit: 10,
        refillRate: 5,
        refillInterval: 1000,
      },
      enabled: true,
    },
  });

  expect(res.status, `expected 200, received: ${JSON.stringify(res, null, 2)}`).toBe(200);

  const found = await h.db.primary.query.keys.findFirst({
    where: (table, { eq }) => eq(table.id, key.id),
    with: {
      ratelimits: true,
    },
  });
  expect(found).toBeDefined();
  expect(found?.name).toEqual("test");
  expect(found?.ownerId).toBeNull();
  expect(found?.meta).toBeNull();
  expect(found?.remaining).toBeNull();
  expect(found!.ratelimits.length).toBe(1);
  expect(found!.ratelimits[0].name).toBe("default");
  expect(found!.ratelimits[0].limit).toBe(10);
  expect(found!.ratelimits[0].duration).toBe(1000);
});

describe("update roles", () => {
  test("creates all missing roles", async (t) => {
    const h = await IntegrationHarness.init(t);
    const root = await h.createRootKey([
      "rbac.*.create_permission",
      "api.*.update_key",
      "rbac.*.add_role_to_key",
      "rbac.*.create_role",
    ]);

    const { keyId } = await h.createKey();

    const name = randomUUID();

    const res = await h.post<V1KeysUpdateKeyRequest, V1KeysUpdateKeyResponse>({
      url: "/v1/keys.updateKey",
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
    const root = await h.createRootKey([
      "rbac.*.create_permission",
      `api.${h.resources.userApi.id}.update_key`,
      "rbac.*.add_role_to_key",
    ]);

    const roles = new Array(3).fill(null).map((_) => ({
      id: newId("test"),
      name: randomUUID(),
      workspaceId: h.resources.userWorkspace.id,
    }));

    await h.db.primary.insert(schema.roles).values(roles);

    const { keyId } = await h.createKey();

    const res = await h.post<V1KeysUpdateKeyRequest, V1KeysUpdateKeyResponse>({
      url: "/v1/keys.updateKey",
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

  test("not desired roles are removed", async (t) => {
    const h = await IntegrationHarness.init(t);
    const root = await h.createRootKey([
      "rbac.*.create_permission",
      "rbac.*.add_role_to_key",
      "rbac.*.remove_role_from_key",
      "api.*.update_key",
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
    const res = await h.post<V1KeysUpdateKeyRequest, V1KeysUpdateKeyResponse>({
      url: "/v1/keys.updateKey",
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
    const root = await h.createRootKey([
      "rbac.*.create_permission",
      "rbac.*.add_role_to_key",
      "api.*.update_key",
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
    const res = await h.post<V1KeysUpdateKeyRequest, V1KeysUpdateKeyResponse>({
      url: "/v1/keys.updateKey",
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
});

describe("update permissions", () => {
  test("creates all missing permissions", async (t) => {
    const h = await IntegrationHarness.init(t);
    const root = await h.createRootKey([
      "rbac.*.create_permission",
      "api.*.update_key",
      "rbac.*.add_permission_to_key",
    ]);

    const { keyId } = await h.createKey();

    const name = randomUUID();

    const res = await h.post<V1KeysUpdateKeyRequest, V1KeysUpdateKeyResponse>({
      url: "/v1/keys.updateKey",
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
    const root = await h.createRootKey([
      "rbac.*.create_permission",
      `api.${h.resources.userApi.id}.update_key`,
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

    const res = await h.post<V1KeysUpdateKeyRequest, V1KeysUpdateKeyResponse>({
      url: "/v1/keys.updateKey",
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
      "rbac.*.add_permission_to_key",
      "rbac.*.remove_permission_from_key",
      "api.*.update_key",
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
    const res = await h.post<V1KeysUpdateKeyRequest, V1KeysUpdateKeyResponse>({
      url: "/v1/keys.updateKey",
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
    const root = await h.createRootKey([
      "rbac.*.create_permission",
      "rbac.*.add_permission_to_key",
      "api.*.update_key",
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
    const res = await h.post<V1KeysUpdateKeyRequest, V1KeysUpdateKeyResponse>({
      url: "/v1/keys.updateKey",
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
});

test("delete expires", async (t) => {
  const h = await IntegrationHarness.init(t);

  const key = {
    id: newId("test"),
    keyAuthId: h.resources.userKeyAuth.id,
    workspaceId: h.resources.userWorkspace.id,
    start: "test",
    name: "test",
    hash: await sha256(new KeyV1({ byteLength: 16 }).toString()),
    createdAtM: Date.now(),
    expires: new Date(Date.now() + 24 * 60 * 60 * 1000),
  };
  await h.db.primary.insert(schema.keys).values(key);
  const root = await h.createRootKey([`api.${h.resources.userApi.id}.update_key`]);

  const res = await h.post<V1KeysUpdateKeyRequest, V1KeysUpdateKeyResponse>({
    url: "/v1/keys.updateKey",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${root.key}`,
    },
    body: {
      keyId: key.id,
      expires: null,
      enabled: true,
    },
  });

  expect(res.status, `expected 200, received: ${JSON.stringify(res, null, 2)}`).toBe(200);

  const found = await h.db.primary.query.keys.findFirst({
    where: (table, { eq }) => eq(table.id, key.id),
  });
  expect(found).toBeDefined();
  expect(found?.name).toEqual("test");
  expect(found?.ownerId).toBeNull();
  expect(found?.meta).toBeNull();
  expect(found?.expires).toBeNull();
});

describe("externalId", () => {
  test("set externalId connects the identity", async (t) => {
    const h = await IntegrationHarness.init(t);

    const root = await h.createRootKey([`api.${h.resources.userApi.id}.update_key`]);

    const key = await h.createKey();
    const externalId = newId("test");

    const res = await h.post<V1KeysUpdateKeyRequest, V1KeysUpdateKeyResponse>({
      url: "/v1/keys.updateKey",
      headers: {
        "Content-Type": "application/json",
        Authorization: `Bearer ${root.key}`,
      },
      body: {
        keyId: key.keyId,
        externalId,
      },
    });

    expect(res.status, `expected 200, received: ${JSON.stringify(res, null, 2)}`).toBe(200);

    const found = await h.db.primary.query.keys.findFirst({
      where: (table, { eq }) => eq(table.id, key.keyId),
      with: {
        identity: true,
      },
    });
    expect(found).toBeDefined();
    expect(found!.identity).toBeDefined();
    expect(found!.identity!.externalId).toBe(externalId);
  });

  test("omitting the field does not disconnect the identity", async (t) => {
    const h = await IntegrationHarness.init(t);

    const root = await h.createRootKey([`api.${h.resources.userApi.id}.update_key`]);

    const identityId = newId("test");
    const externalId = newId("test");
    await h.db.primary.insert(schema.identities).values({
      id: identityId,
      workspaceId: h.resources.userWorkspace.id,
      externalId,
    });
    const key = await h.createKey({ identityId });
    const before = await h.db.primary.query.keys.findFirst({
      where: (table, { eq }) => eq(table.id, key.keyId),
      with: {
        identity: true,
      },
    });
    expect(before?.identity).toBeDefined();

    const res = await h.post<V1KeysUpdateKeyRequest, V1KeysUpdateKeyResponse>({
      url: "/v1/keys.updateKey",
      headers: {
        "Content-Type": "application/json",
        Authorization: `Bearer ${root.key}`,
      },
      body: {
        keyId: key.keyId,
        externalId: undefined,
      },
    });

    expect(res.status, `expected 200, received: ${JSON.stringify(res, null, 2)}`).toBe(200);

    const found = await h.db.primary.query.keys.findFirst({
      where: (table, { eq }) => eq(table.id, key.keyId),
      with: {
        identity: true,
      },
    });
    expect(found).toBeDefined();
    expect(found!.identity).toBeDefined();
    expect(found!.identity!.externalId).toBe(externalId);
  });

  test("set ownerId connects the identity", async (t) => {
    const h = await IntegrationHarness.init(t);

    const root = await h.createRootKey([`api.${h.resources.userApi.id}.update_key`]);

    const key = await h.createKey();
    const ownerId = newId("test");

    const res = await h.post<V1KeysUpdateKeyRequest, V1KeysUpdateKeyResponse>({
      url: "/v1/keys.updateKey",
      headers: {
        "Content-Type": "application/json",
        Authorization: `Bearer ${root.key}`,
      },
      body: {
        keyId: key.keyId,
        ownerId,
      },
    });

    expect(res.status, `expected 200, received: ${JSON.stringify(res, null, 2)}`).toBe(200);

    const found = await h.db.primary.query.keys.findFirst({
      where: (table, { eq }) => eq(table.id, key.keyId),
      with: {
        identity: true,
      },
    });
    expect(found).toBeDefined();
    expect(found!.identity).toBeDefined();
    expect(found!.identity!.externalId).toBe(ownerId);
  });

  test("set externalId=null disconnects the identity", async (t) => {
    const h = await IntegrationHarness.init(t);

    const root = await h.createRootKey([`api.${h.resources.userApi.id}.update_key`]);

    const identityId = newId("test");
    await h.db.primary.insert(schema.identities).values({
      id: identityId,
      workspaceId: h.resources.userWorkspace.id,
      externalId: newId("test"),
    });
    const key = await h.createKey({ identityId });
    const before = await h.db.primary.query.keys.findFirst({
      where: (table, { eq }) => eq(table.id, key.keyId),
      with: {
        identity: true,
      },
    });
    expect(before?.identity).toBeDefined();

    const res = await h.post<V1KeysUpdateKeyRequest, V1KeysUpdateKeyResponse>({
      url: "/v1/keys.updateKey",
      headers: {
        "Content-Type": "application/json",
        Authorization: `Bearer ${root.key}`,
      },
      body: {
        keyId: key.keyId,
        externalId: null,
      },
    });

    expect(res.status, `expected 200, received: ${JSON.stringify(res, null, 2)}`).toBe(200);

    const found = await h.db.primary.query.keys.findFirst({
      where: (table, { eq }) => eq(table.id, key.keyId),
      with: {
        identity: true,
      },
    });
    expect(found).toBeDefined();
    expect(found!.identity).toBeNull();
  });

  test("set ownerId=null disconnects the identity", async (t) => {
    const h = await IntegrationHarness.init(t);

    const root = await h.createRootKey([`api.${h.resources.userApi.id}.update_key`]);

    const identityId = newId("test");
    await h.db.primary.insert(schema.identities).values({
      id: identityId,
      workspaceId: h.resources.userWorkspace.id,
      externalId: newId("test"),
    });
    const key = await h.createKey({ identityId });
    const before = await h.db.primary.query.keys.findFirst({
      where: (table, { eq }) => eq(table.id, key.keyId),
      with: {
        identity: true,
      },
    });
    expect(before?.identity).toBeDefined();

    const res = await h.post<V1KeysUpdateKeyRequest, V1KeysUpdateKeyResponse>({
      url: "/v1/keys.updateKey",
      headers: {
        "Content-Type": "application/json",
        Authorization: `Bearer ${root.key}`,
      },
      body: {
        keyId: key.keyId,
        ownerId: null,
      },
    });

    expect(res.status, `expected 200, received: ${JSON.stringify(res, null, 2)}`).toBe(200);

    const found = await h.db.primary.query.keys.findFirst({
      where: (table, { eq }) => eq(table.id, key.keyId),
      with: {
        identity: true,
      },
    });
    expect(found).toBeDefined();
    expect(found!.identity).toBeNull();
  });
});
test("update should not affect undefined fields", async (t) => {
  const h = await IntegrationHarness.init(t);

  const key = {
    id: newId("test"),
    keyAuthId: h.resources.userKeyAuth.id,
    workspaceId: h.resources.userWorkspace.id,
    start: "test",
    name: "test",
    hash: await sha256(new KeyV1({ byteLength: 16 }).toString()),
    createdAtM: Date.now(),
    ownerId: "ownerId",
    expires: new Date(Date.now() + 60 * 60 * 1000),
  };
  await h.db.primary.insert(schema.keys).values(key);
  const root = await h.createRootKey([`api.${h.resources.userApi.id}.update_key`]);

  const res = await h.post<V1KeysUpdateKeyRequest, V1KeysUpdateKeyResponse>({
    url: "/v1/keys.updateKey",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${root.key}`,
    },
    body: {
      keyId: key.id,
      ownerId: "newOwnerId",
      enabled: true,
    },
  });

  expect(res.status, `expected 200, received: ${JSON.stringify(res, null, 2)}`).toBe(200);

  const found = await h.db.primary.query.keys.findFirst({
    where: (table, { eq }) => eq(table.id, key.id),
    with: {
      ratelimits: true,
    },
  });
  expect(found).toBeDefined();
  expect(found?.name).toEqual("test");
  expect(found?.ownerId).toEqual("newOwnerId");
  expect(found?.meta).toBeNull();
  expect(found?.expires).toEqual(key.expires);
  expect(found?.remaining).toBeNull();
  expect(found!.ratelimits.length).toBe(0);
});

test("update enabled true", async (t) => {
  const h = await IntegrationHarness.init(t);

  const key = {
    id: newId("test"),
    keyAuthId: h.resources.userKeyAuth.id,
    workspaceId: h.resources.userWorkspace.id,
    start: "test",
    name: "test",
    hash: await sha256(new KeyV1({ byteLength: 16 }).toString()),
    createdAtM: Date.now(),
    enabled: false,
  };
  await h.db.primary.insert(schema.keys).values(key);
  const root = await h.createRootKey([`api.${h.resources.userApi.id}.update_key`]);

  const res = await h.post<V1KeysUpdateKeyRequest, V1KeysUpdateKeyResponse>({
    url: "/v1/keys.updateKey",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${root.key}`,
    },
    body: {
      keyId: key.id,
      enabled: true,
    },
  });

  expect(res.status, `expected 200, received: ${JSON.stringify(res, null, 2)}`).toBe(200);

  const found = await h.db.primary.query.keys.findFirst({
    where: (table, { eq }) => eq(table.id, key.id),
  });
  expect(found).toBeDefined();
  expect(found?.name).toEqual("test");
  expect(found?.enabled).toEqual(true);
});

test("update enabled false", async (t) => {
  const h = await IntegrationHarness.init(t);

  const key = {
    id: newId("test"),
    keyAuthId: h.resources.userKeyAuth.id,
    workspaceId: h.resources.userWorkspace.id,
    start: "test",
    name: "test",
    hash: await sha256(new KeyV1({ byteLength: 16 }).toString()),
    createdAtM: Date.now(),
    enabled: true,
  };
  await h.db.primary.insert(schema.keys).values(key);
  const root = await h.createRootKey([`api.${h.resources.userApi.id}.update_key`]);

  const res = await h.post<V1KeysUpdateKeyRequest, V1KeysUpdateKeyResponse>({
    url: "/v1/keys.updateKey",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${root.key}`,
    },
    body: {
      keyId: key.id,
      enabled: false,
    },
  });

  expect(res.status, `expected 200, received: ${JSON.stringify(res, null, 2)}`).toBe(200);

  const found = await h.db.primary.query.keys.findFirst({
    where: (table, { eq }) => eq(table.id, key.id),
  });
  expect(found).toBeDefined();
  expect(found?.name).toEqual("test");
  expect(found?.enabled).toEqual(false);
});

test("omit enabled update", async (t) => {
  const h = await IntegrationHarness.init(t);

  const key = {
    id: newId("test"),
    keyAuthId: h.resources.userKeyAuth.id,
    workspaceId: h.resources.userWorkspace.id,
    start: "test",
    name: "test",
    hash: await sha256(new KeyV1({ byteLength: 16 }).toString()),
    createdAtM: Date.now(),
    enabled: true,
  };
  await h.db.primary.insert(schema.keys).values(key);
  const root = await h.createRootKey([`api.${h.resources.userApi.id}.update_key`]);

  const res = await h.post<V1KeysUpdateKeyRequest, V1KeysUpdateKeyResponse>({
    url: "/v1/keys.updateKey",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${root.key}`,
    },
    body: {
      keyId: key.id,
    },
  });

  expect(res.status, `expected 200, received: ${JSON.stringify(res, null, 2)}`).toBe(200);

  const found = await h.db.primary.query.keys.findFirst({
    where: (table, { eq }) => eq(table.id, key.id),
  });
  expect(found).toBeDefined();
  expect(found?.name).toEqual("test");
  expect(found?.enabled).toEqual(true);
});

test("update ratelimit should not disable it", async (t) => {
  const h = await IntegrationHarness.init(t);

  const root = await h.createRootKey([
    `api.${h.resources.userApi.id}.create_key`,
    `api.${h.resources.userApi.id}.update_key`,
  ]);

  const key = await h.post<V1KeysCreateKeyRequest, V1KeysCreateKeyResponse>({
    url: "/v1/keys.createKey",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${root.key}`,
    },
    body: {
      apiId: h.resources.userApi.id,
      name: "my key",
      ownerId: "team_123",
    },
  });

  expect(key.status).toBe(200);

  const update = await h.post<V1KeysUpdateKeyRequest, V1KeysUpdateKeyResponse>({
    url: "/v1/keys.updateKey",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${root.key}`,
    },
    body: {
      keyId: key.body.keyId,
      name: "Customer X",
      ownerId: "user_123",
      ratelimit: {
        async: true,
        limit: 5,
        duration: 5000,
      },
    },
  });

  expect(update.status).toBe(200);

  const found = await h.db.primary.query.keys.findFirst({
    where: (table, { eq }) => eq(table.id, key.body.keyId),
    with: {
      ratelimits: true,
    },
  });

  expect(found).toBeDefined();
  expect(found!.ratelimits.length).toBe(1);
  expect(found!.ratelimits[0].name).toBe("default");
  expect(found!.ratelimits[0].limit).toBe(5);
  expect(found!.ratelimits[0].duration).toBe(5000);

  const verify = await h.post<V1KeysVerifyKeyRequest, V1KeysVerifyKeyResponse>({
    url: "/v1/keys.verifyKey",
    headers: {
      "Content-Type": "application/json",
    },
    body: {
      apiId: h.resources.userApi.id,
      key: key.body.key,
    },
  });

  expect(verify.status, `expected 200, received: ${JSON.stringify(verify)}`).toBe(200);
  expect(verify.body.ratelimit).toBeDefined();
  expect(verify.body.ratelimit!.limit).toBe(5);
  expect(verify.body.ratelimit!.remaining).toBe(4);
});

describe("When refillDay is omitted.", () => {
  test("should provide default value", async (t) => {
    const h = await IntegrationHarness.init(t);

    const key = {
      id: newId("test"),
      keyAuthId: h.resources.userKeyAuth.id,
      workspaceId: h.resources.userWorkspace.id,
      start: "test",
      name: "test",
      remaining: 10,
      hash: await sha256(new KeyV1({ byteLength: 16 }).toString()),

      createdAtM: Date.now(),
    };
    await h.db.primary.insert(schema.keys).values(key);
    const root = await h.createRootKey([`api.${h.resources.userApi.id}.update_key`]);
    const res = await h.post<V1KeysUpdateKeyRequest, V1KeysUpdateKeyResponse>({
      url: "/v1/keys.updateKey",
      headers: {
        "Content-Type": "application/json",
        Authorization: `Bearer ${root.key}`,
      },
      body: {
        keyId: key.id,
        refill: {
          interval: "monthly",
          amount: 130,
        },
        enabled: true,
      },
    });

    expect(res.status, `expected 200, received: ${JSON.stringify(res, null, 2)}`).toBe(200);

    const found = await h.db.primary.query.keys.findFirst({
      where: (table, { eq }) => eq(table.id, key.id),
    });
    expect(found).toBeDefined();
    expect(found?.remaining).toEqual(10);
    expect(found?.refillAmount).toEqual(130);
    expect(found?.refillDay).toEqual(1);
  });
});

describe("update name", () => {
  test("should not affect ratelimit config", async (t) => {
    const h = await IntegrationHarness.init(t);

    const root = await h.createRootKey([
      `api.${h.resources.userApi.id}.create_key`,
      `api.${h.resources.userApi.id}.update_key`,
    ]);

    const key = await h.post<V1KeysCreateKeyRequest, V1KeysCreateKeyResponse>({
      url: "/v1/keys.createKey",
      headers: {
        "Content-Type": "application/json",
        Authorization: `Bearer ${root.key}`,
      },
      body: {
        apiId: h.resources.userApi.id,
        prefix: "prefix",
        name: "my key",
        ratelimit: {
          async: true,
          limit: 10,
          duration: 1000,
        },
      },
    });

    expect(key.status).toBe(200);

    const update = await h.post<V1KeysUpdateKeyRequest, V1KeysUpdateKeyResponse>({
      url: "/v1/keys.updateKey",
      headers: {
        "Content-Type": "application/json",
        Authorization: `Bearer ${root.key}`,
      },
      body: {
        keyId: key.body.keyId,
        name: "changed",
      },
    });

    expect(update.status).toBe(200);

    const found = await h.db.primary.query.keys.findFirst({
      where: (table, { eq }) => eq(table.id, key.body.keyId),
      with: {
        ratelimits: true,
      },
    });

    expect(found).toBeDefined();
    expect(found!.ratelimits.length).toBe(1);
    expect(found!.ratelimits[0].name).toBe("default");
    expect(found!.ratelimits[0].limit).toBe(10);
    expect(found!.ratelimits[0].duration).toBe(1000);

    const verify = await h.post<V1KeysVerifyKeyRequest, V1KeysVerifyKeyResponse>({
      url: "/v1/keys.verifyKey",
      headers: {
        "Content-Type": "application/json",
      },
      body: {
        apiId: h.resources.userApi.id,
        key: key.body.key,
      },
    });

    expect(verify.status, `expected 200, received: ${JSON.stringify(verify)}`).toBe(200);
    expect(verify.body.ratelimit).toBeDefined();
    expect(verify.body.ratelimit!.limit).toBe(10);
    expect(verify.body.ratelimit!.remaining).toBe(9);
  });
});

describe("update remaining with new credits table", () => {
  test("updates credits table when key has new credits", async (t) => {
    const h = await IntegrationHarness.init(t);

    const keyId = newId("test");
    const key = {
      id: keyId,
      keyAuthId: h.resources.userKeyAuth.id,
      workspaceId: h.resources.userWorkspace.id,
      start: "test",
      name: "test",
      hash: await sha256(new KeyV1({ byteLength: 16 }).toString()),
      createdAtM: Date.now(),
    };
    await h.db.primary.insert(schema.keys).values(key);

    const creditId = newId("credit");
    await h.db.primary.insert(schema.credits).values({
      id: creditId,
      keyId: keyId,
      workspaceId: h.resources.userWorkspace.id,
      remaining: 100,
      createdAt: Date.now(),
      refilledAt: Date.now(),
      identityId: null,
      refillAmount: null,
      refillDay: null,
      updatedAt: null,
    });

    const root = await h.createRootKey([`api.${h.resources.userApi.id}.update_key`]);

    const res = await h.post<V1KeysUpdateKeyRequest, V1KeysUpdateKeyResponse>({
      url: "/v1/keys.updateKey",
      headers: {
        "Content-Type": "application/json",
        Authorization: `Bearer ${root.key}`,
      },
      body: {
        keyId: key.id,
        remaining: 200,
        enabled: true,
      },
    });

    expect(res.status, `expected 200, received: ${JSON.stringify(res, null, 2)}`).toBe(200);

    // Verify the credits table was updated
    const updatedCredit = await h.db.primary.query.credits.findFirst({
      where: (table, { eq }) => eq(table.id, creditId),
    });
    expect(updatedCredit?.remaining).toEqual(200);

    // Verify the keys table was NOT updated (should remain null)
    const updatedKey = await h.db.primary.query.keys.findFirst({
      where: (table, { eq }) => eq(table.id, keyId),
    });
    expect(updatedKey?.remaining).toBeNull();
  });

  test("creates new credits table entry when key has no credits", async (t) => {
    const h = await IntegrationHarness.init(t);

    const keyId = newId("test");
    const key = {
      id: keyId,
      keyAuthId: h.resources.userKeyAuth.id,
      workspaceId: h.resources.userWorkspace.id,
      start: "test",
      name: "test",
      hash: await sha256(new KeyV1({ byteLength: 16 }).toString()),
      createdAtM: Date.now(),
    };
    await h.db.primary.insert(schema.keys).values(key);

    const root = await h.createRootKey([`api.${h.resources.userApi.id}.update_key`]);

    const res = await h.post<V1KeysUpdateKeyRequest, V1KeysUpdateKeyResponse>({
      url: "/v1/keys.updateKey",
      headers: {
        "Content-Type": "application/json",
        Authorization: `Bearer ${root.key}`,
      },
      body: {
        keyId: key.id,
        remaining: 150,
        enabled: true,
      },
    });

    expect(res.status, `expected 200, received: ${JSON.stringify(res, null, 2)}`).toBe(200);

    // Verify a new credits table entry was created
    const credit = await h.db.primary.query.credits.findFirst({
      where: (table, { eq }) => eq(table.keyId, keyId),
    });
    expect(credit).toBeDefined();
    expect(credit?.remaining).toEqual(150);
    expect(credit?.keyId).toEqual(keyId);
    expect(credit?.workspaceId).toEqual(h.resources.userWorkspace.id);

    // Verify the keys table was NOT updated (should remain null)
    const updatedKey = await h.db.primary.query.keys.findFirst({
      where: (table, { eq }) => eq(table.id, keyId),
    });
    expect(updatedKey?.remaining).toBeNull();
  });

  test("updates legacy key.remaining when it exists", async (t) => {
    const h = await IntegrationHarness.init(t);

    const keyId = newId("test");
    const key = {
      id: keyId,
      keyAuthId: h.resources.userKeyAuth.id,
      workspaceId: h.resources.userWorkspace.id,
      start: "test",
      name: "test",
      hash: await sha256(new KeyV1({ byteLength: 16 }).toString()),
      remaining: 100,
      createdAtM: Date.now(),
    };
    await h.db.primary.insert(schema.keys).values(key);

    const root = await h.createRootKey([`api.${h.resources.userApi.id}.update_key`]);

    const res = await h.post<V1KeysUpdateKeyRequest, V1KeysUpdateKeyResponse>({
      url: "/v1/keys.updateKey",
      headers: {
        "Content-Type": "application/json",
        Authorization: `Bearer ${root.key}`,
      },
      body: {
        keyId: key.id,
        remaining: 50,
        enabled: true,
      },
    });

    expect(res.status, `expected 200, received: ${JSON.stringify(res, null, 2)}`).toBe(200);

    // Verify the keys table was updated
    const updatedKey = await h.db.primary.query.keys.findFirst({
      where: (table, { eq }) => eq(table.id, keyId),
    });
    expect(updatedKey?.remaining).toEqual(50);

    // Verify NO credits table entry was created (we should stay in legacy system)
    const credit = await h.db.primary.query.credits.findFirst({
      where: (table, { eq }) => eq(table.keyId, keyId),
    });
    expect(credit).toBeUndefined();
  });

  test("updates refill in credits table when key has new credits", async (t) => {
    const h = await IntegrationHarness.init(t);

    const keyId = newId("test");
    const key = {
      id: keyId,
      keyAuthId: h.resources.userKeyAuth.id,
      workspaceId: h.resources.userWorkspace.id,
      start: "test",
      name: "test",
      hash: await sha256(new KeyV1({ byteLength: 16 }).toString()),
      createdAtM: Date.now(),
    };
    await h.db.primary.insert(schema.keys).values(key);

    const creditId = newId("credit");
    await h.db.primary.insert(schema.credits).values({
      id: creditId,
      keyId: keyId,
      workspaceId: h.resources.userWorkspace.id,
      remaining: 100,
      createdAt: Date.now(),
      refilledAt: Date.now(),
      identityId: null,
      refillAmount: null,
      refillDay: null,
      updatedAt: null,
    });

    const root = await h.createRootKey([`api.${h.resources.userApi.id}.update_key`]);

    const res = await h.post<V1KeysUpdateKeyRequest, V1KeysUpdateKeyResponse>({
      url: "/v1/keys.updateKey",
      headers: {
        "Content-Type": "application/json",
        Authorization: `Bearer ${root.key}`,
      },
      body: {
        keyId: key.id,
        refill: {
          interval: "monthly",
          amount: 500,
          refillDay: 15,
        },
        enabled: true,
      },
    });

    expect(res.status, `expected 200, received: ${JSON.stringify(res, null, 2)}`).toBe(200);

    // Verify the credits table was updated
    const updatedCredit = await h.db.primary.query.credits.findFirst({
      where: (table, { eq }) => eq(table.id, creditId),
    });
    expect(updatedCredit?.refillAmount).toEqual(500);
    expect(updatedCredit?.refillDay).toEqual(15);

    // Verify the keys table was NOT updated
    const updatedKey = await h.db.primary.query.keys.findFirst({
      where: (table, { eq }) => eq(table.id, keyId),
    });
    expect(updatedKey?.refillAmount).toBeNull();
    expect(updatedKey?.refillDay).toBeNull();
  });
});
