import { expect, test } from "bun:test";

import { init } from "@/pkg/global";
import { newApp } from "@/pkg/hono/app";
import { unitTestEnv } from "@/pkg/testutil/env";
import { fetchRoute } from "@/pkg/testutil/request";
import { seed } from "@/pkg/testutil/seed";
import { schema } from "@unkey/db";
import { sha256 } from "@unkey/hash";
import { newId } from "@unkey/id";
import { KeyV1 } from "@unkey/keys";
import {
  V1KeysUpdateKeyRequest,
  V1KeysUpdateKeyResponse,
  registerV1KeysUpdate,
} from "./v1_keys_updateKey";

test("returns 200", async () => {
  const env = unitTestEnv.parse(process.env);
  // @ts-ignore
  init({ env });
  const app = newApp();
  registerV1KeysUpdate(app);

  const r = await seed(env);

  const key = {
    id: newId("key"),
    keyAuthId: r.userKeyAuth.id,
    workspaceId: r.userWorkspace.id,
    start: "test",
    name: "test",
    hash: await sha256(new KeyV1({ byteLength: 16 }).toString()),
    createdAt: new Date(),
  };
  await r.database.insert(schema.keys).values(key);

  const res = await fetchRoute<V1KeysUpdateKeyRequest, V1KeysUpdateKeyResponse>(app, {
    method: "POST",
    url: "/v1/keys.updateKey",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${r.rootKey}`,
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

  expect(res.status).toEqual(200);
});

test("update all", async () => {
  const env = unitTestEnv.parse(process.env);
  // @ts-ignore
  init({ env });
  const app = newApp();
  registerV1KeysUpdate(app);

  const r = await seed(env);

  const key = {
    id: newId("key"),
    keyAuthId: r.userKeyAuth.id,
    workspaceId: r.userWorkspace.id,
    start: "test",
    name: "test",
    hash: await sha256(new KeyV1({ byteLength: 16 }).toString()),
    createdAt: new Date(),
  };
  await r.database.insert(schema.keys).values(key);

  const res = await fetchRoute<V1KeysUpdateKeyRequest, V1KeysUpdateKeyResponse>(app, {
    method: "POST",
    url: "/v1/keys.updateKey",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${r.rootKey}`,
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

  expect(res.status).toEqual(200);

  const found = await r.database.query.keys.findFirst({
    where: (table, { eq }) => eq(table.id, key.id),
  });
  expect(found).toBeDefined();
  expect(found?.name).toEqual("newName");
  expect(found?.ownerId).toEqual("newOwnerId");
  expect(found?.meta).toEqual(JSON.stringify({ new: "meta" }));
  expect(found?.ratelimitType).toEqual("fast");
  expect(found?.ratelimitLimit).toEqual(10);
  expect(found?.ratelimitRefillRate).toEqual(5);
  expect(found?.ratelimitRefillInterval).toEqual(1000);
  expect(found?.remaining).toEqual(0);
});

test("update ratelimit", async () => {
  const env = unitTestEnv.parse(process.env);
  // @ts-ignore
  init({ env });
  const app = newApp();
  registerV1KeysUpdate(app);

  const r = await seed(env);

  const key = {
    id: newId("key"),
    keyAuthId: r.userKeyAuth.id,
    workspaceId: r.userWorkspace.id,
    start: "test",
    name: "test",
    hash: await sha256(new KeyV1({ byteLength: 16 }).toString()),
    createdAt: new Date(),
  };
  await r.database.insert(schema.keys).values(key);

  const res = await fetchRoute<V1KeysUpdateKeyRequest, V1KeysUpdateKeyResponse>(app, {
    method: "POST",
    url: "/v1/keys.updateKey",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${r.rootKey}`,
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

  expect(res.status).toEqual(200);

  const found = await r.database.query.keys.findFirst({
    where: (table, { eq }) => eq(table.id, key.id),
  });
  expect(found).toBeDefined();
  expect(found?.name).toEqual("test");
  expect(found?.ownerId).toBeNull();
  expect(found?.meta).toBeNull();
  expect(found?.ratelimitType).toEqual("fast");
  expect(found?.ratelimitLimit).toEqual(10);
  expect(found?.ratelimitRefillRate).toEqual(5);
  expect(found?.ratelimitRefillInterval).toEqual(1000);
  expect(found?.remaining).toBeNull();
});

test("delete expires", async () => {
  const env = unitTestEnv.parse(process.env);
  // @ts-ignore
  init({ env });
  const app = newApp();
  registerV1KeysUpdate(app);

  const r = await seed(env);

  const key = {
    id: newId("key"),
    keyAuthId: r.userKeyAuth.id,
    workspaceId: r.userWorkspace.id,
    start: "test",
    name: "test",
    hash: await sha256(new KeyV1({ byteLength: 16 }).toString()),
    createdAt: new Date(),
    expires: new Date(Date.now() + 24 * 60 * 60 * 1000),
  };
  await r.database.insert(schema.keys).values(key);

  const res = await fetchRoute<V1KeysUpdateKeyRequest, V1KeysUpdateKeyResponse>(app, {
    method: "POST",
    url: "/v1/keys.updateKey",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${r.rootKey}`,
    },
    body: {
      keyId: key.id,
      expires: null,
      enabled: true,
    },
  });

  expect(res.status).toEqual(200);

  const found = await r.database.query.keys.findFirst({
    where: (table, { eq }) => eq(table.id, key.id),
  });
  expect(found).toBeDefined();
  expect(found?.name).toEqual("test");
  expect(found?.ownerId).toBeNull();
  expect(found?.meta).toBeNull();
  expect(found?.expires).toBeNull();
});

test("update should not affect undefined fields", async () => {
  const env = unitTestEnv.parse(process.env);
  // @ts-ignore
  init({ env });
  const app = newApp();
  registerV1KeysUpdate(app);

  const r = await seed(env);

  const key = {
    id: newId("key"),
    keyAuthId: r.userKeyAuth.id,
    workspaceId: r.userWorkspace.id,
    start: "test",
    name: "test",
    hash: await sha256(new KeyV1({ byteLength: 16 }).toString()),
    createdAt: new Date(),
    ownerId: "ownerId",
    expires: new Date(Date.now() + 60 * 60 * 1000),
  };
  await r.database.insert(schema.keys).values(key);

  const res = await fetchRoute<V1KeysUpdateKeyRequest, V1KeysUpdateKeyResponse>(app, {
    method: "POST",
    url: "/v1/keys.updateKey",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${r.rootKey}`,
    },
    body: {
      keyId: key.id,
      ownerId: "newOwnerId",
      enabled: true,
    },
  });

  expect(res.status).toEqual(200);

  const found = await r.database.query.keys.findFirst({
    where: (table, { eq }) => eq(table.id, key.id),
  });
  expect(found).toBeDefined();
  expect(found?.name).toEqual("test");
  expect(found?.ownerId).toEqual("newOwnerId");
  expect(found?.meta).toBeNull();
  expect(found?.expires).toEqual(key.expires);
  expect(found?.ratelimitType).toBeNull();
  expect(found?.ratelimitLimit).toBeNull();
  expect(found?.ratelimitRefillRate).toBeNull();
  expect(found?.ratelimitRefillInterval).toBeNull();
  expect(found?.remaining).toBeNull();
});

test("update enabled true", async () => {
  const env = unitTestEnv.parse(process.env);
  // @ts-ignore
  init({ env });
  const app = newApp();
  registerV1KeysUpdate(app);

  const r = await seed(env);

  const key = {
    id: newId("key"),
    keyAuthId: r.userKeyAuth.id,
    workspaceId: r.userWorkspace.id,
    start: "test",
    name: "test",
    hash: await sha256(new KeyV1({ byteLength: 16 }).toString()),
    createdAt: new Date(),
    enabled: false,
  };
  await r.database.insert(schema.keys).values(key);

  const res = await fetchRoute<V1KeysUpdateKeyRequest, V1KeysUpdateKeyResponse>(app, {
    method: "POST",
    url: "/v1/keys.updateKey",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${r.rootKey}`,
    },
    body: {
      keyId: key.id,
      enabled: true,
    },
  });

  expect(res.status).toEqual(200);

  const found = await r.database.query.keys.findFirst({
    where: (table, { eq }) => eq(table.id, key.id),
  });
  expect(found).toBeDefined();
  expect(found?.name).toEqual("test");
  expect(found?.enabled).toEqual(true);
});

test("update enabled false", async () => {
  const env = unitTestEnv.parse(process.env);
  // @ts-ignore
  init({ env });
  const app = newApp();
  registerV1KeysUpdate(app);

  const r = await seed(env);

  const key = {
    id: newId("key"),
    keyAuthId: r.userKeyAuth.id,
    workspaceId: r.userWorkspace.id,
    start: "test",
    name: "test",
    hash: await sha256(new KeyV1({ byteLength: 16 }).toString()),
    createdAt: new Date(),
    enabled: true,
  };
  await r.database.insert(schema.keys).values(key);

  const res = await fetchRoute<V1KeysUpdateKeyRequest, V1KeysUpdateKeyResponse>(app, {
    method: "POST",
    url: "/v1/keys.updateKey",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${r.rootKey}`,
    },
    body: {
      keyId: key.id,
      enabled: false,
    },
  });

  expect(res.status).toEqual(200);

  const found = await r.database.query.keys.findFirst({
    where: (table, { eq }) => eq(table.id, key.id),
  });
  expect(found).toBeDefined();
  expect(found?.name).toEqual("test");
  expect(found?.enabled).toEqual(false);
});

test("omit enabled update", async () => {
  const env = unitTestEnv.parse(process.env);
  // @ts-ignore
  init({ env });
  const app = newApp();
  registerV1KeysUpdate(app);

  const r = await seed(env);

  const key = {
    id: newId("key"),
    keyAuthId: r.userKeyAuth.id,
    workspaceId: r.userWorkspace.id,
    start: "test",
    name: "test",
    hash: await sha256(new KeyV1({ byteLength: 16 }).toString()),
    createdAt: new Date(),
    enabled: true,
  };
  await r.database.insert(schema.keys).values(key);

  const res = await fetchRoute<V1KeysUpdateKeyRequest, V1KeysUpdateKeyResponse>(app, {
    method: "POST",
    url: "/v1/keys.updateKey",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${r.rootKey}`,
    },
    body: {
      keyId: key.id,
    },
  });

  expect(res.status).toEqual(200);

  const found = await r.database.query.keys.findFirst({
    where: (table, { eq }) => eq(table.id, key.id),
  });
  expect(found).toBeDefined();
  expect(found?.name).toEqual("test");
  expect(found?.enabled).toEqual(true);
});
