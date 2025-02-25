import { schema } from "@unkey/db";
import { sha256 } from "@unkey/hash";
import { newId } from "@unkey/id";
import { KeyV1 } from "@unkey/keys";
import { IntegrationHarness } from "src/pkg/testutil/integration-harness";

import { expect, test } from "vitest";
import type { V1KeysGetVerificationsResponse } from "./v1_keys_getVerifications";

test("returns an empty verifications array", async (t) => {
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
  const root = await h.createRootKey([`api.${h.resources.userApi.id}.read_key`]);
  const res = await h.get<V1KeysGetVerificationsResponse>({
    url: `/v1/keys.getVerifications?keyId=${keyId}`,
    headers: {
      Authorization: `Bearer ${root.key}`,
    },
  });

  expect(res.status, `expected 200, received: ${JSON.stringify(res, null, 2)}`).toBe(200);
  expect(res.body).toEqual({
    verifications: [],
  });
});

test("ownerId works too", async (t) => {
  const h = await IntegrationHarness.init(t);
  const ownerId = crypto.randomUUID();
  const keyIds = [newId("test"), newId("test"), newId("test")];
  for (const keyId of keyIds) {
    const key = new KeyV1({ prefix: "test", byteLength: 16 }).toString();
    await h.db.primary.insert(schema.keys).values({
      id: keyId,
      keyAuthId: h.resources.userKeyAuth.id,
      hash: await sha256(key),
      start: key.slice(0, 8),
      workspaceId: h.resources.userWorkspace.id,
      createdAtM: Date.now(),
      ownerId,
    });
  }
  const root = await h.createRootKey([`api.${h.resources.userApi.id}.read_key`]);

  const res = await h.get<V1KeysGetVerificationsResponse>({
    url: `/v1/keys.getVerifications?ownerId=${ownerId}`,
    headers: {
      Authorization: `Bearer ${root.key}`,
    },
  });

  expect(res.status, `expected 200, received: ${JSON.stringify(res, null, 2)}`).toBe(200);
  expect(res.body).toEqual({
    verifications: [],
  });
});
