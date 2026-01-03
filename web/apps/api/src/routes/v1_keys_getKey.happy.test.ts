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
