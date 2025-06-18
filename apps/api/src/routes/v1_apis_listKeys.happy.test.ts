import { expect, test } from "vitest";

import { eq, schema } from "@unkey/db";
import { sha256 } from "@unkey/hash";
import { newId } from "@unkey/id";
import { KeyV1 } from "@unkey/keys";
import { IntegrationHarness } from "src/pkg/testutil/integration-harness";

import { randomUUID } from "node:crypto";
import { revalidateKeyCount } from "@/pkg/util/revalidate_key_count";
import type { V1ApisListKeysResponse } from "./v1_apis_listKeys";
import type {
  V1MigrationsCreateKeysRequest,
  V1MigrationsCreateKeysResponse,
} from "./v1_migrations_createKey";

test("get api", async (t) => {
  const h = await IntegrationHarness.init(t);
  const keyIds = new Array(10).fill(0).map(() => newId("test"));
  for (let i = 0; i < keyIds.length; i++) {
    const key = new KeyV1({ prefix: "test", byteLength: 16 }).toString();
    await h.db.primary.insert(schema.keys).values({
      id: keyIds[i],
      keyAuthId: h.resources.userKeyAuth.id,
      hash: await sha256(key),
      start: key.slice(0, 8),
      workspaceId: h.resources.userWorkspace.id,
      createdAtM: Date.now(),
    });
  }
  await revalidateKeyCount(h.db.primary, h.resources.userKeyAuth.id);
  const root = await h.createRootKey([
    `api.${h.resources.userApi.id}.read_api`,
    `api.${h.resources.userApi.id}.read_key`,
  ]);

  const res = await h.get<V1ApisListKeysResponse>({
    url: `/v1/apis.listKeys?apiId=${h.resources.userApi.id}`,
    headers: {
      Authorization: `Bearer ${root.key}`,
    },
  });

  expect(res.status, `expected 200, received: ${JSON.stringify(res, null, 2)}`).toBe(200);
  expect(res.body.total).toBeGreaterThanOrEqual(keyIds.length);
  expect(res.body.keys.length).toBeGreaterThanOrEqual(keyIds.length);
  expect(res.body.keys.length).toBeLessThanOrEqual(100); //  default page size
});

test("returns enabled", async (t) => {
  const h = await IntegrationHarness.init(t);

  const keys = new Array(10)
    .fill(0)
    .map(() => ({ id: newId("test"), enabled: Math.random() > 0.5 }));
  for (let i = 0; i < keys.length; i++) {
    const key = new KeyV1({ prefix: "test", byteLength: 16 }).toString();
    await h.db.primary.insert(schema.keys).values({
      id: keys[i].id,
      keyAuthId: h.resources.userKeyAuth.id,
      hash: await sha256(key),
      start: key.slice(0, 8),
      workspaceId: h.resources.userWorkspace.id,
      enabled: keys[i].enabled,
      createdAtM: Date.now(),
    });
  }

  await revalidateKeyCount(h.db.primary, h.resources.userKeyAuth.id);
  const root = await h.createRootKey([
    `api.${h.resources.userApi.id}.read_api`,
    `api.${h.resources.userApi.id}.read_key`,
  ]);

  const res = await h.get<V1ApisListKeysResponse>({
    url: `/v1/apis.listKeys?apiId=${h.resources.userApi.id}`,
    headers: {
      Authorization: `Bearer ${root.key}`,
    },
  });

  expect(res.status, `expected 200, received: ${JSON.stringify(res, null, 2)}`).toBe(200);
  expect(res.body.total).toBeGreaterThanOrEqual(keys.length);
  expect(res.body.keys.length).toBeGreaterThanOrEqual(keys.length);
  expect(res.body.keys.length).toBeLessThanOrEqual(100); //  default page size

  for (const key of res.body.keys) {
    const found = keys.find((k) => k.id === key.id);
    expect(found).toBeDefined();
    expect(found!.enabled).toBe(key.enabled);
  }
});

test("returns identity", async (t) => {
  const h = await IntegrationHarness.init(t);

  const identity = {
    id: newId("identity"),
    externalId: randomUUID(),
    workspaceId: h.resources.userWorkspace.id,
  };
  await h.db.primary.insert(schema.identities).values(identity);

  const keyIds = new Array(10).fill(0).map(() => newId("test"));
  for (let i = 0; i < keyIds.length; i++) {
    const key = new KeyV1({ prefix: "test", byteLength: 16 }).toString();
    await h.db.primary.insert(schema.keys).values({
      id: keyIds[i],
      keyAuthId: h.resources.userKeyAuth.id,
      hash: await sha256(key),
      start: key.slice(0, 8),
      workspaceId: h.resources.userWorkspace.id,
      identityId: identity.id,
      createdAtM: Date.now(),
    });
  }

  await revalidateKeyCount(h.db.primary, h.resources.userKeyAuth.id);
  const root = await h.createRootKey([
    `api.${h.resources.userApi.id}.read_api`,
    `api.${h.resources.userApi.id}.read_key`,
  ]);

  const res = await h.get<V1ApisListKeysResponse>({
    url: `/v1/apis.listKeys?apiId=${h.resources.userApi.id}`,
    headers: {
      Authorization: `Bearer ${root.key}`,
    },
  });

  expect(res.status, `expected 200, received: ${JSON.stringify(res, null, 2)}`).toBe(200);
  expect(res.body.total).toBeGreaterThanOrEqual(keyIds.length);
  expect(res.body.keys.length).toBeGreaterThanOrEqual(keyIds.length);
  expect(res.body.keys.length).toBeLessThanOrEqual(100); //  default page size
  for (const key of res.body.keys) {
    expect(key.identity).toBeDefined();
    expect(key.identity!.id).toEqual(identity.id);
    expect(key.identity!.externalId).toEqual(identity.externalId);
  }
});

test("filter by ownerId", async (t) => {
  const h = await IntegrationHarness.init(t);
  const ownerId = crypto.randomUUID();
  const keyIds = new Array(10).fill(0).map(() => newId("test"));
  for (let i = 0; i < keyIds.length; i++) {
    const key = new KeyV1({ prefix: "test", byteLength: 16 }).toString();
    await h.db.primary.insert(schema.keys).values({
      id: keyIds[i],
      keyAuthId: h.resources.userKeyAuth.id,
      hash: await sha256(key),
      start: key.slice(0, 8),
      workspaceId: h.resources.userWorkspace.id,
      createdAtM: Date.now(),
      ownerId: i % 2 === 0 ? ownerId : undefined,
    });
  }
  await revalidateKeyCount(h.db.primary, h.resources.userKeyAuth.id);

  const root = await h.createRootKey([
    `api.${h.resources.userApi.id}.read_api`,
    `api.${h.resources.userApi.id}.read_key`,
  ]);

  const res = await h.get<V1ApisListKeysResponse>({
    url: `/v1/apis.listKeys?apiId=${h.resources.userApi.id}&ownerId=${ownerId}`,
    headers: {
      Authorization: `Bearer ${root.key}`,
    },
  });

  expect(res.status, `expected 200, received: ${JSON.stringify(res, null, 2)}`).toBe(200);
  expect(res.body.total).toBeGreaterThanOrEqual(keyIds.length);
  expect(res.body.keys).toHaveLength(5);
});

test("filter by externalId", async (t) => {
  const h = await IntegrationHarness.init(t);
  const identity = {
    id: newId("test"),
    externalId: randomUUID(),
    workspaceId: h.resources.userWorkspace.id,
  };
  await h.db.primary.insert(schema.identities).values(identity);
  const keyIds = new Array(10).fill(0).map(() => newId("test"));
  for (let i = 0; i < keyIds.length; i++) {
    const key = new KeyV1({ prefix: "test", byteLength: 16 }).toString();
    await h.db.primary.insert(schema.keys).values({
      id: keyIds[i],
      keyAuthId: h.resources.userKeyAuth.id,
      hash: await sha256(key),
      start: key.slice(0, 8),
      workspaceId: h.resources.userWorkspace.id,
      createdAtM: Date.now(),
      identityId: i % 2 === 0 ? identity.id : undefined,
    });
  }
  await revalidateKeyCount(h.db.primary, h.resources.userKeyAuth.id);

  const root = await h.createRootKey([
    `api.${h.resources.userApi.id}.read_api`,
    `api.${h.resources.userApi.id}.read_key`,
  ]);

  const res = await h.get<V1ApisListKeysResponse>({
    url: `/v1/apis.listKeys?apiId=${h.resources.userApi.id}&externalId=${identity.externalId}`,
    headers: {
      Authorization: `Bearer ${root.key}`,
    },
  });

  expect(res.status, `expected 200, received: ${JSON.stringify(res, null, 2)}`).toBe(200);
  expect(res.body.total).toBeGreaterThanOrEqual(keyIds.length);
  expect(res.body.keys).toHaveLength(5);
});

test("returns roles and permissions", async (t) => {
  const h = await IntegrationHarness.init(t);

  const roleId = newId("test");
  const roleName = randomUUID();
  const permissionId = newId("test");
  const permissionName = randomUUID();

  const keyId = newId("test");
  const key = new KeyV1({ prefix: "test", byteLength: 16 }).toString();
  await h.db.primary.insert(schema.keys).values({
    id: keyId,
    keyAuthId: h.resources.userKeyAuth.id,
    hash: await sha256(key),
    start: key.slice(0, 8),
    workspaceId: h.resources.userWorkspace.id,
    createdAtM: Date.now(),
  });
  await h.db.primary.insert(schema.roles).values({
    id: roleId,
    name: roleName,
    workspaceId: h.resources.userWorkspace.id,
  });
  await h.db.primary.insert(schema.permissions).values({
    id: permissionId,
    name: permissionName,
    slug: permissionName,
    workspaceId: h.resources.userWorkspace.id,
  });
  await h.db.primary.insert(schema.rolesPermissions).values({
    permissionId,
    roleId,
    workspaceId: h.resources.userWorkspace.id,
  });
  await h.db.primary.insert(schema.keysRoles).values({
    keyId,
    roleId,
    workspaceId: h.resources.userWorkspace.id,
  });

  const root = await h.createRootKey([
    `api.${h.resources.userApi.id}.read_api`,
    `api.${h.resources.userApi.id}.read_key`,
  ]);

  const res = await h.get<V1ApisListKeysResponse>({
    url: `/v1/apis.listKeys?apiId=${h.resources.userApi.id}`,
    headers: {
      Authorization: `Bearer ${root.key}`,
    },
  });

  expect(res.status, `expected 200, received: ${JSON.stringify(res, null, 2)}`).toBe(200);
  expect(res.body.keys).toHaveLength(1);
  const found = res.body.keys[0];
  expect(found.roles).toEqual([roleName]);
  expect(found.permissions).toEqual([permissionName]);
});

test("with limit", async (t) => {
  const h = await IntegrationHarness.init(t);
  const keyIds = new Array(10).fill(0).map(() => newId("test"));
  for (let i = 0; i < keyIds.length; i++) {
    const key = new KeyV1({ prefix: "test", byteLength: 16 }).toString();
    await h.db.primary.insert(schema.keys).values({
      id: keyIds[i],
      keyAuthId: h.resources.userKeyAuth.id,
      hash: await sha256(key),
      start: key.slice(0, 8),
      workspaceId: h.resources.userWorkspace.id,
      createdAtM: Date.now(),
    });
  }
  await revalidateKeyCount(h.db.primary, h.resources.userKeyAuth.id);

  const root = await h.createRootKey([
    `api.${h.resources.userApi.id}.read_api`,
    `api.${h.resources.userApi.id}.read_key`,
  ]);

  const res = await h.get<V1ApisListKeysResponse>({
    url: `/v1/apis.listKeys?apiId=${h.resources.userApi.id}&limit=2`,
    headers: {
      Authorization: `Bearer ${root.key}`,
    },
  });
  expect(res.status, `expected 200, received: ${JSON.stringify(res, null, 2)}`).toBe(200);
  expect(res.body.total).toBeGreaterThanOrEqual(keyIds.length);
  expect(res.body.keys).toHaveLength(2);
}, 10_000);

test("with cursor", async (t) => {
  const h = await IntegrationHarness.init(t);
  const keyIds = new Array(10).fill(0).map(() => newId("test"));
  for (let i = 0; i < keyIds.length; i++) {
    const key = new KeyV1({ prefix: "test", byteLength: 16 }).toString();
    await h.db.primary.insert(schema.keys).values({
      id: keyIds[i],
      keyAuthId: h.resources.userKeyAuth.id,
      hash: await sha256(key),
      start: key.slice(0, 8),
      workspaceId: h.resources.userWorkspace.id,
      createdAtM: Date.now(),
    });
  }
  await revalidateKeyCount(h.db.primary, h.resources.userKeyAuth.id);

  const root = await h.createRootKey([
    `api.${h.resources.userApi.id}.read_api`,
    `api.${h.resources.userApi.id}.read_key`,
  ]);
  const res1 = await h.get<V1ApisListKeysResponse>({
    url: `/v1/apis.listKeys?apiId=${h.resources.userApi.id}&limit=2`,
    headers: {
      Authorization: `Bearer ${root.key}`,
    },
  });
  expect(res1.status).toEqual(200);
  expect(res1.body.cursor).toBeDefined();

  const res2 = await h.get<V1ApisListKeysResponse>({
    url: `/v1/apis.listKeys?apiId=${h.resources.userApi.id}&limit=3&cursor=${res1.body.cursor}`,
    headers: {
      Authorization: `Bearer ${root.key}`,
    },
  });

  expect(res2.status).toEqual(200);
  const found = new Set<string>();
  for (const key of res1.body.keys) {
    found.add(key.id);
  }
  for (const key of res2.body.keys) {
    found.add(key.id);
  }
  expect(found.size).toEqual(5);
});

test("retrieves a key in plain text", async (t) => {
  const h = await IntegrationHarness.init(t);

  const root = await h.createRootKey([
    `api.${h.resources.userApi.id}.read_api`,
    `api.${h.resources.userApi.id}.create_key`,
    `api.${h.resources.userApi.id}.read_key`,
    `api.${h.resources.userApi.id}.decrypt_key`,
  ]);

  await h.db.primary
    .update(schema.keyAuth)
    .set({
      storeEncryptedKeys: true,
    })
    .where(eq(schema.keyAuth.id, h.resources.userKeyAuth.id));

  const key = new KeyV1({ byteLength: 16, prefix: "test" }).toString();
  const hash = await sha256(key);

  const res = await h.post<V1MigrationsCreateKeysRequest, V1MigrationsCreateKeysResponse>({
    url: "/v1/migrations.createKeys",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${root.key}`,
    },
    body: [
      {
        plaintext: key,
        apiId: h.resources.userApi.id,
      },
    ],
  });
  expect(res.status, `expected 200, received: ${JSON.stringify(res, null, 2)}`).toBe(200);
  expect(res.body.keyIds.length).toEqual(1);

  const found = await h.db.primary.query.keys.findFirst({
    where: (table, { eq }) => eq(table.id, res.body.keyIds[0]),
  });

  expect(found!.hash).toEqual(hash);

  const listKeysRes = await h.get<V1ApisListKeysResponse>({
    url: `/v1/apis.listKeys?apiId=${h.resources.userApi.id}&decrypt=true`,
    headers: {
      Authorization: `Bearer ${root.key}`,
    },
  });

  expect(listKeysRes.status).toBe(200);
  expect(listKeysRes.body.keys.at(0)?.plaintext).toEqual(key);
});
