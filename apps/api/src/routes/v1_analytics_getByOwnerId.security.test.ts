import { randomInt, randomUUID } from "crypto";
import { RouteHarness } from "@/pkg/testutil/route-harness";
import { runSharedRoleTests } from "@/pkg/testutil/test_route_roles";
import { schema } from "@unkey/db";
import { sha256 } from "@unkey/hash";
import { newId } from "@unkey/id";
import { KeyV1 } from "@unkey/keys";
import { afterEach, beforeEach, describe, expect, test } from "vitest";
import {
  type V1KeysGetVerificationsResponse,
  registerV1AnalyticsGetByOwnerId,
} from "./v1_analytics_getByOwnerId";

let h: RouteHarness;
beforeEach(async () => {
  h = new RouteHarness();
  h.useRoutes(registerV1AnalyticsGetByOwnerId);
  await h.seed();
});
afterEach(async () => {
  await h.teardown();
});

runSharedRoleTests({
  registerHandler: registerV1AnalyticsGetByOwnerId,
  prepareRequest: async (rh) => {
    const keyId = newId("key");
    const key = new KeyV1({ prefix: "test", byteLength: 16 }).toString();
    const ownerId = `testOwnerId-${randomInt(100000, 999999)}`;
    await rh.db.insert(schema.keys).values({
      id: keyId,
      keyAuthId: rh.resources.userKeyAuth.id,
      hash: await sha256(key),
      start: key.slice(0, 8),
      workspaceId: rh.resources.userWorkspace.id,
      ownerId: ownerId,
      createdAt: new Date(),
    });
    return {
      method: "GET",
      url: `/v1/analytics.getByOwnerId?ownerId=${ownerId}`,
    };
  },
});

describe("correct roles", () => {
  test.each([
    { name: "legacy", roles: ["*"] },
    { name: "legacy and more", roles: ["*", randomUUID()] },
    { name: "wildcard api", roles: ["api.*.read_key"] },

    {
      name: "wildcard and more",
      roles: ["api.*.read_key", "api.*.read_api", randomUUID()],
    },
    {
      name: "specific apiId",
      roles: [(apiId: string) => `api.${apiId}.read_key`],
    },
    {
      name: "specific apiId and more",
      roles: [(apiId: string) => `api.${apiId}.read_key`, randomUUID()],
    },
  ])("$name", async ({ roles }) => {
    const keyId = newId("key");
    const key = new KeyV1({ prefix: "test", byteLength: 16 }).toString();
    const ownerId = `testOwnerId-${randomInt(100000, 999999)}`;
    await h.db.insert(schema.keys).values({
      id: keyId,
      keyAuthId: h.resources.userKeyAuth.id,
      hash: await sha256(key),
      start: key.slice(0, 8),
      workspaceId: h.resources.userWorkspace.id,
      ownerId: ownerId,
      createdAt: new Date(),
    });
    const root = await h.createRootKey(
      roles.map((role) => (typeof role === "string" ? role : role(h.resources.userApi.id))),
    );

    const res = await h.get<V1KeysGetVerificationsResponse>({
      url: `/v1/analytics.getByOwnerId?ownerId=${ownerId}`,
      headers: {
        Authorization: `Bearer ${root.key}`,
      },
    });
    expect(res.status).toEqual(200);
  });
});

test("cannot read keys from a different workspace", async () => {
  await h.seed();
  h.useRoutes(registerV1AnalyticsGetByOwnerId);

  const workspaceId = newId("workspace");
  await h.db.insert(schema.workspaces).values({
    id: workspaceId,
    tenantId: randomUUID(),
    name: randomUUID(),
    features: {},
    betaFeatures: {},
  });

  const keyAuthId = newId("keyAuth");
  await h.db.insert(schema.keyAuth).values({
    id: keyAuthId,
    workspaceId,
  });

  const apiId = newId("api");
  await h.db.insert(schema.apis).values({
    id: apiId,
    name: randomUUID(),
    workspaceId,
    authType: "key",
    keyAuthId,
  });

  const keyId = newId("key");
  const key = new KeyV1({ prefix: "test", byteLength: 16 }).toString();
  const ownerId = `testOwnerId-${randomInt(100000, 999999)}`;
  await h.db.insert(schema.keys).values({
    id: keyId,
    keyAuthId: keyAuthId,
    hash: await sha256(key),
    start: key.slice(0, 8),
    workspaceId,
    ownerId: ownerId,
    createdAt: new Date(),
  });

  const keyId2 = newId("key");
  const key2 = new KeyV1({ prefix: "test", byteLength: 16 }).toString();
  const ownerId2 = `testOwnerId-${randomInt(100000, 999999)}`;
  await h.db.insert(schema.keys).values({
    id: keyId2,
    keyAuthId: h.resources.userKeyAuth.id,
    hash: await sha256(key2),
    start: key2.slice(0, 8),
    workspaceId: h.resources.userWorkspace.id,
    ownerId: ownerId2,
    createdAt: new Date(),
  });

  const root = await h.createRootKey([`api.${apiId}.read_key`]);

  const res = await h.get<V1KeysGetVerificationsResponse>({
    url: `/v1/analytics.getByOwnerId?ownerId=${ownerId2}`,
    headers: {
      Authorization: `Bearer ${root.key}`,
    },
  });
  expect(res.status).toEqual(403);
  expect(res.body).toMatchObject({
    error: {},
  });
});
