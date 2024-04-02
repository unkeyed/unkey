import { expect, test } from "vitest";

import { schema } from "@unkey/db";
import { sha256 } from "@unkey/hash";
import { newId } from "@unkey/id";
import { KeyV1 } from "@unkey/keys";
import { RouteHarness } from "src/pkg/testutil/route-harness";
import type { V1ApisListKeysResponse } from "./v1_apis_listKeys";

test("get api", async (t) => {
  const h = await RouteHarness.init(t);
  const keyIds = new Array(10).fill(0).map(() => newId("key"));
  for (let i = 0; i < keyIds.length; i++) {
    const key = new KeyV1({ prefix: "test", byteLength: 16 }).toString();
    await h.db.insert(schema.keys).values({
      id: keyIds[i],
      keyAuthId: h.resources.userKeyAuth.id,
      hash: await sha256(key),
      start: key.slice(0, 8),
      workspaceId: h.resources.userWorkspace.id,
      createdAt: new Date(),
    });
  }
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

  expect(res.status).toEqual(200);
  expect(res.body.total).toBeGreaterThanOrEqual(keyIds.length);
  expect(res.body.keys.length).toBeGreaterThanOrEqual(keyIds.length);
  expect(res.body.keys.length).toBeLessThanOrEqual(100); //  default page size
});

test("filter by ownerId", async (t) => {
  const h = await RouteHarness.init(t);
  const ownerId = crypto.randomUUID();
  const keyIds = new Array(10).fill(0).map(() => newId("key"));
  for (let i = 0; i < keyIds.length; i++) {
    const key = new KeyV1({ prefix: "test", byteLength: 16 }).toString();
    await h.db.insert(schema.keys).values({
      id: keyIds[i],
      keyAuthId: h.resources.userKeyAuth.id,
      hash: await sha256(key),
      start: key.slice(0, 8),
      workspaceId: h.resources.userWorkspace.id,
      createdAt: new Date(),
      ownerId: i % 2 === 0 ? ownerId : undefined,
    });
  }

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

  expect(res.status).toEqual(200);
  expect(res.body.total).toBeGreaterThanOrEqual(keyIds.length);
  expect(res.body.keys).toHaveLength(5);
});

test("with limit", async (t) => {
  const h = await RouteHarness.init(t);
  const keyIds = new Array(10).fill(0).map(() => newId("key"));
  for (let i = 0; i < keyIds.length; i++) {
    const key = new KeyV1({ prefix: "test", byteLength: 16 }).toString();
    await h.db.insert(schema.keys).values({
      id: keyIds[i],
      keyAuthId: h.resources.userKeyAuth.id,
      hash: await sha256(key),
      start: key.slice(0, 8),
      workspaceId: h.resources.userWorkspace.id,
      createdAt: new Date(),
    });
  }

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
  expect(res.status).toEqual(200);
  expect(res.body.total).toBeGreaterThanOrEqual(keyIds.length);
  expect(res.body.keys).toHaveLength(2);
}, 10_000);

test("with cursor", async (t) => {
  const h = await RouteHarness.init(t);
  const keyIds = new Array(10).fill(0).map(() => newId("key"));
  for (let i = 0; i < keyIds.length; i++) {
    const key = new KeyV1({ prefix: "test", byteLength: 16 }).toString();
    await h.db.insert(schema.keys).values({
      id: keyIds[i],
      keyAuthId: h.resources.userKeyAuth.id,
      hash: await sha256(key),
      start: key.slice(0, 8),
      workspaceId: h.resources.userWorkspace.id,
      createdAt: new Date(),
    });
  }

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
