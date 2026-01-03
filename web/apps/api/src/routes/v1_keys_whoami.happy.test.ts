import { schema } from "@unkey/db";
import { sha256 } from "@unkey/hash";
import { newId } from "@unkey/id";
import { KeyV1 } from "@unkey/keys";
import { IntegrationHarness } from "src/pkg/testutil/integration-harness";

import { randomUUID } from "node:crypto";
import { expect, test } from "vitest";
import type { V1KeysWhoAmIRequest, V1KeysWhoAmIResponse } from "./v1_keys_whoami";

test("returns 200", async (t) => {
  const h = await IntegrationHarness.init(t);
  const root = await h.createRootKey(["api.*.read_key"]);

  const key = new KeyV1({ byteLength: 16 }).toString();
  const hash = await sha256(key);
  const meta = JSON.stringify({ hello: "world" });

  const keySchema = {
    id: newId("test"),
    keyAuthId: h.resources.userKeyAuth.id,
    workspaceId: h.resources.userWorkspace.id,
    start: "test",
    name: "test",
    remaining: 100,
    enabled: true,
    environment: "test",
    hash: hash,
    meta: meta,
    createdAtM: Date.now(),
  };
  await h.db.primary.insert(schema.keys).values(keySchema);

  const res = await h.post<V1KeysWhoAmIRequest, V1KeysWhoAmIResponse>({
    url: "/v1/keys.whoami",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${root.key}`,
    },
    body: {
      key: key,
    },
  });
  expect(res.status, `expected 200, received: ${JSON.stringify(res, null, 2)}`).toBe(200);

  expect(res.body.id).toEqual(keySchema.id);
  expect(res.body.name).toEqual(keySchema.name);
  expect(res.body.remaining).toEqual(keySchema.remaining);
  expect(res.body.name).toEqual(keySchema.name);
  expect(res.body.meta).toEqual(JSON.parse(keySchema.meta));
  expect(res.body.enabled).toEqual(keySchema.enabled);
  expect(res.body.environment).toEqual(keySchema.environment);
});

test("returns identity", async (t) => {
  const h = await IntegrationHarness.init(t);

  const identity = {
    id: newId("identity"),
    externalId: randomUUID(),
    workspaceId: h.resources.userWorkspace.id,
  };
  await h.db.primary.insert(schema.identities).values(identity);

  const { key } = await h.createKey({ identityId: identity.id });
  const root = await h.createRootKey([
    `api.${h.resources.userApi.id}.read_api`,
    `api.${h.resources.userApi.id}.read_key`,
  ]);

  const res = await h.post<V1KeysWhoAmIRequest, V1KeysWhoAmIResponse>({
    url: "/v1/keys.whoami",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${root.key}`,
    },
    body: {
      key: key,
    },
  });

  expect(res.status, `expected 200, received: ${JSON.stringify(res, null, 2)}`).toBe(200);
  expect(res.body.identity).toBeDefined();
  expect(res.body.identity!.id).toEqual(identity.id);
  expect(res.body.identity!.externalId).toEqual(identity.externalId);
});
