import { schema } from "@unkey/db";
import { sha256 } from "@unkey/hash";
import { newId } from "@unkey/id";
import { KeyV1 } from "@unkey/keys";
import { IntegrationHarness } from "src/pkg/testutil/integration-harness";

import { randomUUID } from "node:crypto";
import { expect, test } from "vitest";
import type { V1KeysGetKeyResponse } from "./v1_keys_getKey";

test("returns 200", async (t) => {
  const h = await IntegrationHarness.init(t);
  const root = await h.createRootKey(["api.*.read_key"]);
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

  const res = await h.get<V1KeysGetKeyResponse>({
    url: `/v1/keys.getKey?keyId=${key.id}`,
    headers: {
      Authorization: `Bearer ${root.key}`,
    },
  });
  expect(res.status, `expected 200, received: ${JSON.stringify(res, null, 2)}`).toBe(200);

  expect(res.body.id).toEqual(key.id);
  expect(res.body.apiId).toEqual(h.resources.userApi.id);
  expect(res.body.workspaceId).toEqual(key.workspaceId);
  expect(res.body.name).toEqual(key.name);
  expect(res.body.start).toEqual(key.start);
  expect(res.body.createdAt).toEqual(key.createdAtM);
});

test("returns identity", async (t) => {
  const h = await IntegrationHarness.init(t);

  const identity = {
    id: newId("identity"),
    externalId: randomUUID(),
    workspaceId: h.resources.userWorkspace.id,
  };
  await h.db.primary.insert(schema.identities).values(identity);

  const key = await h.createKey({ identityId: identity.id });
  const root = await h.createRootKey([
    `api.${h.resources.userApi.id}.read_api`,
    `api.${h.resources.userApi.id}.read_key`,
  ]);

  const res = await h.get<V1KeysGetKeyResponse>({
    url: `/v1/keys.getKey?keyId=${key.keyId}`,
    headers: {
      Authorization: `Bearer ${root.key}`,
    },
  });

  expect(res.status, `expected 200, received: ${JSON.stringify(res, null, 2)}`).toBe(200);
  expect(res.body.identity).toBeDefined();
  expect(res.body.identity!.id).toEqual(identity.id);
  expect(res.body.identity!.externalId).toEqual(identity.externalId);
});

test("returns credits from new credits table", async (t) => {
  const h = await IntegrationHarness.init(t);
  const root = await h.createRootKey(["api.*.read_key"]);

  const keyId = newId("test");
  await h.db.primary.insert(schema.keys).values({
    id: keyId,
    keyAuthId: h.resources.userKeyAuth.id,
    workspaceId: h.resources.userWorkspace.id,
    start: "test",
    hash: await sha256(new KeyV1({ byteLength: 16 }).toString()),
    createdAtM: Date.now(),
  });

  // Create credits in new table with monthly refill
  const creditId = newId("credit");
  await h.db.primary.insert(schema.credits).values({
    id: creditId,
    keyId,
    workspaceId: h.resources.userWorkspace.id,
    remaining: 5000,
    refillAmount: 2000,
    refillDay: 10, // monthly refill
    createdAt: Date.now(),
    refilledAt: Date.now(),
    identityId: null,
    updatedAt: null,
  });

  const res = await h.get<V1KeysGetKeyResponse>({
    url: `/v1/keys.getKey?keyId=${keyId}`,
    headers: {
      Authorization: `Bearer ${root.key}`,
    },
  });

  expect(res.status, `expected 200, received: ${JSON.stringify(res, null, 2)}`).toBe(200);
  expect(res.body.remaining).toBe(5000);
  expect(res.body.refill).toBeDefined();
  expect(res.body.refill!.interval).toBe("monthly");
  expect(res.body.refill!.amount).toBe(2000);
  expect(res.body.refill!.refillDay).toBe(10);
});

test("returns credits from legacy remaining field when no credits table entry", async (t) => {
  const h = await IntegrationHarness.init(t);
  const root = await h.createRootKey(["api.*.read_key"]);

  const keyId = newId("test");
  await h.db.primary.insert(schema.keys).values({
    id: keyId,
    keyAuthId: h.resources.userKeyAuth.id,
    workspaceId: h.resources.userWorkspace.id,
    start: "test",
    hash: await sha256(new KeyV1({ byteLength: 16 }).toString()),
    remaining: 800,
    refillAmount: 100,
    refillDay: null, // daily refill
    createdAtM: Date.now(),
  });

  const res = await h.get<V1KeysGetKeyResponse>({
    url: `/v1/keys.getKey?keyId=${keyId}`,
    headers: {
      Authorization: `Bearer ${root.key}`,
    },
  });

  expect(res.status, `expected 200, received: ${JSON.stringify(res, null, 2)}`).toBe(200);
  expect(res.body.remaining).toBe(800);
  expect(res.body.refill).toBeDefined();
  expect(res.body.refill!.interval).toBe("daily");
  expect(res.body.refill!.amount).toBe(100);
});

test("prefers new credits table over legacy remaining field", async (t) => {
  const h = await IntegrationHarness.init(t);
  const root = await h.createRootKey(["api.*.read_key"]);

  const keyId = newId("test");
  // Create key with legacy credits (should be ignored)
  await h.db.primary.insert(schema.keys).values({
    id: keyId,
    keyAuthId: h.resources.userKeyAuth.id,
    workspaceId: h.resources.userWorkspace.id,
    start: "test",
    hash: await sha256(new KeyV1({ byteLength: 16 }).toString()),
    remaining: 123, // This should be ignored
    refillAmount: 50, // This should be ignored
    createdAtM: Date.now(),
  });

  // Create credits in new table (should take precedence)
  const creditId = newId("credit");
  await h.db.primary.insert(schema.credits).values({
    id: creditId,
    keyId,
    workspaceId: h.resources.userWorkspace.id,
    remaining: 7000,
    refillAmount: 3000,
    refillDay: null, // daily refill
    createdAt: Date.now(),
    refilledAt: Date.now(),
    identityId: null,
    updatedAt: null,
  });

  const res = await h.get<V1KeysGetKeyResponse>({
    url: `/v1/keys.getKey?keyId=${keyId}`,
    headers: {
      Authorization: `Bearer ${root.key}`,
    },
  });

  expect(res.status, `expected 200, received: ${JSON.stringify(res, null, 2)}`).toBe(200);
  // Should use values from credits table, not legacy fields
  expect(res.body.remaining).toBe(7000);
  expect(res.body.refill).toBeDefined();
  expect(res.body.refill!.amount).toBe(3000);
});
