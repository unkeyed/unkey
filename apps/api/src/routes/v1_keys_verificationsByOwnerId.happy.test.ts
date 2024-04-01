import { schema } from "@unkey/db";
import { sha256 } from "@unkey/hash";
import { newId } from "@unkey/id";
import { KeyV1 } from "@unkey/keys";
import { RouteHarness } from "src/pkg/testutil/route-harness";
import { expect, test } from "vitest";
import type { V1AnalyticsGetVerificationsResponse } from "./v1_keys_verificationsByOwnerId";

test("returns an empty verifications with OwnerId", async (t) => {
  const h = await RouteHarness.init(t);
  const keyId = newId("key");
  const ownerId = crypto.randomUUID();
  const key = new KeyV1({ prefix: "test", byteLength: 16 }).toString();
  await h.db.insert(schema.keys).values({
    id: keyId,
    keyAuthId: h.resources.userKeyAuth.id,
    hash: await sha256(key),
    start: key.slice(0, 8),
    workspaceId: h.resources.userWorkspace.id,
    createdAt: new Date(),
    ownerId,
  });
  const root = await h.createRootKey([
    `api.${h.resources.userApi.id}.read_key`,
    `api.${h.resources.unkeyApi}.read_api`,
  ]);
  const res = await h.get<V1AnalyticsGetVerificationsResponse>({
    url: `/v1/keys.verificationsByOwnerId?ownerId=${ownerId}`,
    headers: {
      Authorization: `Bearer ${root.key}`,
    },
  });

  expect(res.status).toEqual(200);
  expect(res.body).toEqual({
    ownerId,
    apis: [
      {
        apiId: h.resources.userApi.id,
        apiName: h.resources.userApi.name,
        keys: [keyId],
      },
    ],
    verifications: [],
  });
});

test("with apiId worked", async (t) => {
  const h = await RouteHarness.init(t);
  const ownerId = crypto.randomUUID();
  const keyIds = [newId("key"), newId("key"), newId("key")];
  for (const keyId of keyIds) {
    const key = new KeyV1({ prefix: "test", byteLength: 16 }).toString();
    await h.db.insert(schema.keys).values({
      id: keyId,
      keyAuthId: h.resources.userKeyAuth.id,
      hash: await sha256(key),
      start: key.slice(0, 8),
      workspaceId: h.resources.userWorkspace.id,
      createdAt: new Date(),
      ownerId,
    });
  }
  const root = await h.createRootKey([`api.${h.resources.userApi.id}.read_key`]);

  const res = await h.get<V1AnalyticsGetVerificationsResponse>({
    url: `/v1/keys.verificationsByOwnerId?ownerId=${ownerId}&apiId=${h.resources.userApi.id}`,
    headers: {
      Authorization: `Bearer ${root.key}`,
    },
  });
  keyIds.sort();
  expect(res.status).toEqual(200);
  expect(res.body).toEqual({
    verifications: [],
  });
});
