import { expect, test } from "vitest";

import { IntegrationHarness } from "@/pkg/testutil/integration-harness";

import { schema } from "@unkey/db";
import { sha256 } from "@unkey/hash";
import { newId } from "@unkey/id";
import { KeyV1 } from "@unkey/keys";
import type {
  V1KeysUpdateRemainingRequest,
  V1KeysUpdateRemainingResponse,
} from "./v1_keys_updateRemaining";

test("increment", async (t) => {
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

  const root = await h.createRootKey(["api.*.update_key"]);
  const res = await h.post<V1KeysUpdateRemainingRequest, V1KeysUpdateRemainingResponse>({
    url: "/v1/keys.updateRemaining",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${root.key}`,
    },
    body: {
      keyId: key.id,
      op: "increment",
      value: 10,
    },
  });

  expect(res.status, `expected 200, received: ${JSON.stringify(res, null, 2)}`).toBe(200);
  expect(res.body.remaining).toEqual(110);

  // Verify legacy keys.remaining was updated
  const updatedKey = await h.db.primary.query.keys.findFirst({
    where: (table, { eq }) => eq(table.id, keyId),
    with: {
      credits: true,
    },
  });
  expect(updatedKey?.remaining).toEqual(110);
  expect(updatedKey?.credits).toBeNull();
});

test("decrement", async (t) => {
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
  const root = await h.createRootKey(["api.*.update_key"]);

  const res = await h.post<V1KeysUpdateRemainingRequest, V1KeysUpdateRemainingResponse>({
    url: "/v1/keys.updateRemaining",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${root.key}`,
    },
    body: {
      keyId: key.id,
      op: "decrement",
      value: 10,
    },
  });

  expect(res.status, `expected 200, received: ${JSON.stringify(res, null, 2)}`).toBe(200);
  expect(res.body.remaining).toEqual(90);

  // Verify legacy keys.remaining was updated
  const updatedKey = await h.db.primary.query.keys.findFirst({
    where: (table, { eq }) => eq(table.id, keyId),
    with: {
      credits: true,
    },
  });
  expect(updatedKey?.remaining).toEqual(90);
  expect(updatedKey?.credits).toBeNull();
});

test("set", async (t) => {
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
  const root = await h.createRootKey(["api.*.update_key"]);

  const res = await h.post<V1KeysUpdateRemainingRequest, V1KeysUpdateRemainingResponse>({
    url: "/v1/keys.updateRemaining",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${root.key}`,
    },
    body: {
      keyId: key.id,
      op: "set",
      value: 10,
    },
  });

  expect(res.status, `expected 200, received: ${JSON.stringify(res, null, 2)}`).toBe(200);
  expect(res.body.remaining).toEqual(10);

  // Verify legacy keys.remaining was updated
  const updatedKey = await h.db.primary.query.keys.findFirst({
    where: (table, { eq }) => eq(table.id, keyId),
    with: {
      credits: true,
    },
  });
  expect(updatedKey?.remaining).toEqual(10);
  expect(updatedKey?.credits).toBeNull();
});

test("invalid operation", async (t) => {
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
  const root = await h.createRootKey(["api.*.update_key"]);

  const res = await h.post<V1KeysUpdateRemainingRequest, V1KeysUpdateRemainingResponse>({
    url: "/v1/keys.updateRemaining",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${root.key}`,
    },
    body: {
      keyId: key.id,
      // @ts-ignore This is an invalid operation
      op: "XXX",
      value: 10,
    },
  });

  expect(res.status).toEqual(400);
});

test("increment with new credits table", async (t) => {
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

  const root = await h.createRootKey(["api.*.update_key"]);
  const res = await h.post<V1KeysUpdateRemainingRequest, V1KeysUpdateRemainingResponse>({
    url: "/v1/keys.updateRemaining",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${root.key}`,
    },
    body: {
      keyId: key.id,
      op: "increment",
      value: 10,
    },
  });

  expect(res.status, `expected 200, received: ${JSON.stringify(res, null, 2)}`).toBe(200);
  expect(res.body.remaining).toEqual(110);

  // Verify the credits table was updated
  const updatedCredit = await h.db.primary.query.credits.findFirst({
    where: (table, { eq }) => eq(table.id, creditId),
  });
  expect(updatedCredit?.remaining).toEqual(110);

  // Verify the keys table was NOT updated (should remain null)
  const updatedKey = await h.db.primary.query.keys.findFirst({
    where: (table, { eq }) => eq(table.id, keyId),
  });
  expect(updatedKey?.remaining).toBeNull();
});

test("decrement with new credits table", async (t) => {
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

  const root = await h.createRootKey(["api.*.update_key"]);
  const res = await h.post<V1KeysUpdateRemainingRequest, V1KeysUpdateRemainingResponse>({
    url: "/v1/keys.updateRemaining",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${root.key}`,
    },
    body: {
      keyId: key.id,
      op: "decrement",
      value: 10,
    },
  });

  expect(res.status, `expected 200, received: ${JSON.stringify(res, null, 2)}`).toBe(200);
  expect(res.body.remaining).toEqual(90);

  // Verify the credits table was updated
  const updatedCredit = await h.db.primary.query.credits.findFirst({
    where: (table, { eq }) => eq(table.id, creditId),
  });
  expect(updatedCredit?.remaining).toEqual(90);

  // Verify the keys table was NOT updated (should remain null)
  const updatedKey = await h.db.primary.query.keys.findFirst({
    where: (table, { eq }) => eq(table.id, keyId),
  });
  expect(updatedKey?.remaining).toBeNull();
});

test("set operation creates new credits table entry when key has no credits", async (t) => {
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

  const root = await h.createRootKey(["api.*.update_key"]);
  const res = await h.post<V1KeysUpdateRemainingRequest, V1KeysUpdateRemainingResponse>({
    url: "/v1/keys.updateRemaining",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${root.key}`,
    },
    body: {
      keyId: key.id,
      op: "set",
      value: 50,
    },
  });

  expect(res.status, `expected 200, received: ${JSON.stringify(res, null, 2)}`).toBe(200);
  expect(res.body.remaining).toEqual(50);

  // Verify a new credits table entry was created
  const credit = await h.db.primary.query.credits.findFirst({
    where: (table, { eq }) => eq(table.keyId, keyId),
  });
  expect(credit).toBeDefined();
  expect(credit?.remaining).toEqual(50);
  expect(credit?.keyId).toEqual(keyId);
  expect(credit?.workspaceId).toEqual(h.resources.userWorkspace.id);

  // Verify the keys table was NOT updated (should remain null)
  const updatedKey = await h.db.primary.query.keys.findFirst({
    where: (table, { eq }) => eq(table.id, keyId),
  });
  expect(updatedKey?.remaining).toBeNull();
});

test("set operation updates legacy key.remaining when it exists", async (t) => {
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

  const root = await h.createRootKey(["api.*.update_key"]);
  const res = await h.post<V1KeysUpdateRemainingRequest, V1KeysUpdateRemainingResponse>({
    url: "/v1/keys.updateRemaining",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${root.key}`,
    },
    body: {
      keyId: key.id,
      op: "set",
      value: 50,
    },
  });

  expect(res.status, `expected 200, received: ${JSON.stringify(res, null, 2)}`).toBe(200);
  expect(res.body.remaining).toEqual(50);

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
