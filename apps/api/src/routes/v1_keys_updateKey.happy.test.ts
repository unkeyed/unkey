import { expect, test } from "vitest";

import { schema } from "@unkey/db";
import { sha256 } from "@unkey/hash";
import { newId } from "@unkey/id";
import { KeyV1 } from "@unkey/keys";
import { IntegrationHarness } from "src/pkg/testutil/integration-harness";

import type { V1KeysUpdateKeyRequest, V1KeysUpdateKeyResponse } from "./v1_keys_updateKey";

test("returns 200", async (t) => {
  const h = await IntegrationHarness.init(t);

  const key = {
    id: newId("key"),
    keyAuthId: h.resources.userKeyAuth.id,
    workspaceId: h.resources.userWorkspace.id,
    start: "test",
    name: "test",
    hash: await sha256(new KeyV1({ byteLength: 16 }).toString()),
    createdAt: new Date(),
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

  expect(res.status, `expected 200, received: ${JSON.stringify(res)}`).toBe(200);
});

test("update all", async (t) => {
  const h = await IntegrationHarness.init(t);

  const key = {
    id: newId("key"),
    keyAuthId: h.resources.userKeyAuth.id,
    workspaceId: h.resources.userWorkspace.id,
    start: "test",
    name: "test",
    hash: await sha256(new KeyV1({ byteLength: 16 }).toString()),
    createdAt: new Date(),
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

  expect(res.status, `expected 200, received: ${JSON.stringify(res)}`).toBe(200);

  const found = await h.db.readonly.query.keys.findFirst({
    where: (table, { eq }) => eq(table.id, key.id),
  });
  expect(found).toBeDefined();
  expect(found?.name).toEqual("newName");
  expect(found?.ownerId).toEqual("newOwnerId");
  expect(found?.meta).toEqual(JSON.stringify({ new: "meta" }));
  expect(found?.ratelimitAsync).toEqual(true);
  expect(found?.ratelimitLimit).toEqual(10);
  expect(found?.ratelimitAsync).toEqual(1000);
  expect(found?.remaining).toEqual(0);
});

test("update ratelimit", async (t) => {
  const h = await IntegrationHarness.init(t);

  const key = {
    id: newId("key"),
    keyAuthId: h.resources.userKeyAuth.id,
    workspaceId: h.resources.userWorkspace.id,
    start: "test",
    name: "test",
    hash: await sha256(new KeyV1({ byteLength: 16 }).toString()),
    createdAt: new Date(),
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

  expect(res.status, `expected 200, received: ${JSON.stringify(res)}`).toBe(200);

  const found = await h.db.readonly.query.keys.findFirst({
    where: (table, { eq }) => eq(table.id, key.id),
  });
  expect(found).toBeDefined();
  expect(found?.name).toEqual("test");
  expect(found?.ownerId).toBeNull();
  expect(found?.meta).toBeNull();
  expect(found?.ratelimitAsync).toEqual(true);
  expect(found?.ratelimitLimit).toEqual(10);
  expect(found?.ratelimitDuration).toEqual(1000);
  expect(found?.remaining).toBeNull();
});

test("delete expires", async (t) => {
  const h = await IntegrationHarness.init(t);

  const key = {
    id: newId("key"),
    keyAuthId: h.resources.userKeyAuth.id,
    workspaceId: h.resources.userWorkspace.id,
    start: "test",
    name: "test",
    hash: await sha256(new KeyV1({ byteLength: 16 }).toString()),
    createdAt: new Date(),
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

  expect(res.status, `expected 200, received: ${JSON.stringify(res)}`).toBe(200);

  const found = await h.db.readonly.query.keys.findFirst({
    where: (table, { eq }) => eq(table.id, key.id),
  });
  expect(found).toBeDefined();
  expect(found?.name).toEqual("test");
  expect(found?.ownerId).toBeNull();
  expect(found?.meta).toBeNull();
  expect(found?.expires).toBeNull();
});

test("update should not affect undefined fields", async (t) => {
  const h = await IntegrationHarness.init(t);

  const key = {
    id: newId("key"),
    keyAuthId: h.resources.userKeyAuth.id,
    workspaceId: h.resources.userWorkspace.id,
    start: "test",
    name: "test",
    hash: await sha256(new KeyV1({ byteLength: 16 }).toString()),
    createdAt: new Date(),
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

  expect(res.status, `expected 200, received: ${JSON.stringify(res)}`).toBe(200);

  const found = await h.db.readonly.query.keys.findFirst({
    where: (table, { eq }) => eq(table.id, key.id),
  });
  expect(found).toBeDefined();
  expect(found?.name).toEqual("test");
  expect(found?.ownerId).toEqual("newOwnerId");
  expect(found?.meta).toBeNull();
  expect(found?.expires).toEqual(key.expires);
  expect(found?.ratelimitAsync).toBeNull();
  expect(found?.ratelimitLimit).toBeNull();
  expect(found?.ratelimitDuration).toBeNull();
  expect(found?.remaining).toBeNull();
});

test("update enabled true", async (t) => {
  const h = await IntegrationHarness.init(t);

  const key = {
    id: newId("key"),
    keyAuthId: h.resources.userKeyAuth.id,
    workspaceId: h.resources.userWorkspace.id,
    start: "test",
    name: "test",
    hash: await sha256(new KeyV1({ byteLength: 16 }).toString()),
    createdAt: new Date(),
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

  expect(res.status, `expected 200, received: ${JSON.stringify(res)}`).toBe(200);

  const found = await h.db.readonly.query.keys.findFirst({
    where: (table, { eq }) => eq(table.id, key.id),
  });
  expect(found).toBeDefined();
  expect(found?.name).toEqual("test");
  expect(found?.enabled).toEqual(true);
});

test("update enabled false", async (t) => {
  const h = await IntegrationHarness.init(t);

  const key = {
    id: newId("key"),
    keyAuthId: h.resources.userKeyAuth.id,
    workspaceId: h.resources.userWorkspace.id,
    start: "test",
    name: "test",
    hash: await sha256(new KeyV1({ byteLength: 16 }).toString()),
    createdAt: new Date(),
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

  expect(res.status, `expected 200, received: ${JSON.stringify(res)}`).toBe(200);

  const found = await h.db.readonly.query.keys.findFirst({
    where: (table, { eq }) => eq(table.id, key.id),
  });
  expect(found).toBeDefined();
  expect(found?.name).toEqual("test");
  expect(found?.enabled).toEqual(false);
});

test("omit enabled update", async (t) => {
  const h = await IntegrationHarness.init(t);

  const key = {
    id: newId("key"),
    keyAuthId: h.resources.userKeyAuth.id,
    workspaceId: h.resources.userWorkspace.id,
    start: "test",
    name: "test",
    hash: await sha256(new KeyV1({ byteLength: 16 }).toString()),
    createdAt: new Date(),
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

  expect(res.status, `expected 200, received: ${JSON.stringify(res)}`).toBe(200);

  const found = await h.db.readonly.query.keys.findFirst({
    where: (table, { eq }) => eq(table.id, key.id),
  });
  expect(found).toBeDefined();
  expect(found?.name).toEqual("test");
  expect(found?.enabled).toEqual(true);
});
