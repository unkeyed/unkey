import { expect, test } from "vitest";

import { newId } from "@unkey/id";
import { IntegrationHarness } from "src/pkg/testutil/integration-harness";

import type { ErrorResponse } from "@/pkg/errors";
import { schema } from "@unkey/db";
import { sha256 } from "@unkey/hash";
import { KeyV1 } from "@unkey/keys";
import type { V1KeysUpdateKeyRequest, V1KeysUpdateKeyResponse } from "./v1_keys_updateKey";

test("when the key does not exist", async (t) => {
  const h = await IntegrationHarness.init(t);
  const keyId = newId("test");

  const root = await h.createRootKey([`api.${h.resources.userApi.id}.update_key`]);

  const res = await h.post<V1KeysUpdateKeyRequest, V1KeysUpdateKeyResponse>({
    url: "/v1/keys.updateKey",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${root.key}`,
    },
    body: {
      keyId,
      enabled: false,
    },
  });
  expect(res.status).toEqual(404);
  expect(res.body).toMatchObject({
    error: {
      code: "NOT_FOUND",
      docs: "https://unkey.dev/docs/api-reference/errors/code/NOT_FOUND",
      message: `key ${keyId} not found`,
    },
  });
});
test("reject invalid refill config", async (t) => {
  const h = await IntegrationHarness.init(t);
  const keyId = newId("test");
  const root = await h.createRootKey([`api.${h.resources.userApi.id}.update_key`]);
  /* The code snippet is making a POST request to the "/v1/keys.createKey" endpoint with the specified headers. It is using the `h.post` method from the `Harness` instance to send the request. The generic types `<V1KeysCreateKeyRequest, V1KeysCreateKeyResponse>` specify the request payload and response types respectively. */
  const key = {
    id: keyId,
    keyAuthId: h.resources.userKeyAuth.id,
    workspaceId: h.resources.userWorkspace.id,
    start: "test",
    name: "test",
    remaining: 10,
    hash: await sha256(new KeyV1({ byteLength: 16 }).toString()),

    createdAtM: Date.now(),
  };
  await h.db.primary.insert(schema.keys).values(key);

  const res = await h.post<V1KeysUpdateKeyRequest, ErrorResponse>({
    url: "/v1/keys.updateKey",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${root.key}`,
    },
    body: {
      keyId,
      remaining: 10,
      refill: {
        amount: 100,
        refillDay: 4,
        interval: "daily",
      },
    },
  });
  expect(res.status).toEqual(400);
  expect(res.body).toMatchObject({
    error: {
      code: "BAD_REQUEST",
      docs: "https://unkey.dev/docs/api-reference/errors/code/BAD_REQUEST",
      message: "Cannot set 'refillDay' if 'interval' is 'daily'",
    },
  });
});

test("when the key has been deleted", async (t) => {
  const h = await IntegrationHarness.init(t);
  const key = {
    id: newId("test"),
    keyAuthId: h.resources.userKeyAuth.id,
    workspaceId: h.resources.userWorkspace.id,
    start: "test",
    name: "test",
    hash: await sha256(new KeyV1({ byteLength: 16 }).toString()),
    createdAtM: Date.now(),
    deletedAtM: Date.now(),
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
  expect(res.status, `Expected 404, got: ${JSON.stringify(res)}`).toEqual(404);
  expect(res.body).toMatchObject({
    error: {
      code: "NOT_FOUND",
      docs: "https://unkey.dev/docs/api-reference/errors/code/NOT_FOUND",
      message: `key ${key.id} not found`,
    },
  });
});
