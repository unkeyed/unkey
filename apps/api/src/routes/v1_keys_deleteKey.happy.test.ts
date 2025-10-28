import { expect, test } from "vitest";

import { schema } from "@unkey/db";
import { sha256 } from "@unkey/hash";
import { newId } from "@unkey/id";
import { KeyV1 } from "@unkey/keys";

import { IntegrationHarness } from "src/pkg/testutil/integration-harness";

import type { V1KeysDeleteKeyRequest, V1KeysDeleteKeyResponse } from "./v1_keys_deleteKey";

test("soft deletes key", async (t) => {
  const h = await IntegrationHarness.init(t);
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

  const root = await h.createRootKey([`api.${h.resources.userApi.id}.delete_key`]);
  const res = await h.post<V1KeysDeleteKeyRequest, V1KeysDeleteKeyResponse>({
    url: "/v1/keys.deleteKey",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${root.key}`,
    },
    body: {
      keyId,
    },
  });

  expect(res.status, `expected 200, received: ${JSON.stringify(res, null, 2)}`).toBe(200);

  const found = await h.db.primary.query.keys.findFirst({
    where: (table, { eq }) => eq(table.id, keyId),
  });
  expect(found).toBeDefined();
  expect(found!.deletedAtM).toBeDefined();
  expect(found!.deletedAtM! - Date.now()).toBeLessThan(10_000); // 10s play});
});

test("hard deletes key", async (t) => {
  const h = await IntegrationHarness.init(t);
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

  const root = await h.createRootKey([`api.${h.resources.userApi.id}.delete_key`]);
  const res = await h.post<V1KeysDeleteKeyRequest, V1KeysDeleteKeyResponse>({
    url: "/v1/keys.deleteKey",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${root.key}`,
    },
    body: {
      keyId,
      permanent: true,
    },
  });

  expect(res.status, `expected 200, received: ${JSON.stringify(res, null, 2)}`).toBe(200);

  const found = await h.db.primary.query.keys.findFirst({
    where: (table, { eq }) => eq(table.id, keyId),
  });
  expect(found).toBeUndefined();
});

test("permanent delete removes credits from credits table", async (t) => {
  const h = await IntegrationHarness.init(t);
  const keyId = newId("test");
  const key = new KeyV1({ prefix: "test", byteLength: 16 }).toString();

  // Create key
  await h.db.primary.insert(schema.keys).values({
    id: keyId,
    keyAuthId: h.resources.userKeyAuth.id,
    hash: await sha256(key),
    start: key.slice(0, 8),
    workspaceId: h.resources.userWorkspace.id,
    createdAtM: Date.now(),
  });

  // Create credits in new table
  const creditId = newId("credit");
  await h.db.primary.insert(schema.credits).values({
    id: creditId,
    keyId,
    workspaceId: h.resources.userWorkspace.id,
    remaining: 1000,
    refillAmount: null,
    refillDay: null,
    createdAt: Date.now(),
    refilledAt: Date.now(),
    identityId: null,
    updatedAt: null,
  });

  // Verify credits exist
  const creditsBefore = await h.db.primary.query.credits.findFirst({
    where: (table, { eq }) => eq(table.keyId, keyId),
  });
  expect(creditsBefore).toBeDefined();
  expect(creditsBefore!.id).toBe(creditId);

  const root = await h.createRootKey([`api.${h.resources.userApi.id}.delete_key`]);
  const res = await h.post<V1KeysDeleteKeyRequest, V1KeysDeleteKeyResponse>({
    url: "/v1/keys.deleteKey",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${root.key}`,
    },
    body: {
      keyId,
      permanent: true,
    },
  });

  expect(res.status, `expected 200, received: ${JSON.stringify(res, null, 2)}`).toBe(200);

  // Verify key is deleted
  const foundKey = await h.db.primary.query.keys.findFirst({
    where: (table, { eq }) => eq(table.id, keyId),
  });
  expect(foundKey).toBeUndefined();

  // Verify credits are also deleted
  const creditsAfter = await h.db.primary.query.credits.findFirst({
    where: (table, { eq }) => eq(table.keyId, keyId),
  });
  expect(creditsAfter).toBeUndefined();
});
