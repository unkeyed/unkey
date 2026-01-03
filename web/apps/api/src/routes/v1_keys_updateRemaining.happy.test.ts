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
      op: "increment",
      value: 10,
    },
  });

  expect(res.status, `expected 200, received: ${JSON.stringify(res, null, 2)}`).toBe(200);
  expect(res.body.remaining).toEqual(110);
});

test("decrement", async (t) => {
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
      op: "decrement",
      value: 10,
    },
  });

  expect(res.status, `expected 200, received: ${JSON.stringify(res, null, 2)}`).toBe(200);
  expect(res.body.remaining).toEqual(90);
});

test("set", async (t) => {
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
      op: "set",
      value: 10,
    },
  });

  expect(res.status, `expected 200, received: ${JSON.stringify(res, null, 2)}`).toBe(200);
  expect(res.body.remaining).toEqual(10);
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
