import { expect, test } from "vitest";

import { schema } from "@unkey/db";
import { newId } from "@unkey/id";
import { IntegrationHarness } from "src/pkg/testutil/integration-harness";

import { randomUUID } from "node:crypto";
import type { V1IdentitiesListIdentitiesResponse } from "./v1_identities_listIdentities";

test("get identities", async (t) => {
  const h = await IntegrationHarness.init(t);
  const identityIds = new Array(10).fill(0).map(() => newId("test"));
  for (let i = 0; i < identityIds.length; i++) {
    await h.db.primary.insert(schema.identities).values({
      id: identityIds[i],
      externalId: randomUUID(),
      workspaceId: h.resources.userWorkspace.id,
    });
  }
  const root = await h.createRootKey(["identity.*.read_identity"]);

  const res = await h.get<V1IdentitiesListIdentitiesResponse>({
    url: "/v1/identities.listIdentities",
    headers: {
      Authorization: `Bearer ${root.key}`,
    },
  });

  expect(res.status, `expected 200, received: ${JSON.stringify(res, null, 2)}`).toBe(200);
  expect(res.body.total).toBeGreaterThanOrEqual(identityIds.length);
  expect(res.body.identities.length).toBeGreaterThanOrEqual(identityIds.length);
  expect(res.body.identities.length).toBeLessThanOrEqual(100); //  default page size
});

test("filter by environment", async (t) => {
  const h = await IntegrationHarness.init(t);
  const environment = crypto.randomUUID();
  const identityIds = new Array(10).fill(0).map(() => newId("test"));
  for (let i = 0; i < identityIds.length; i++) {
    await h.db.primary.insert(schema.identities).values({
      id: identityIds[i],
      externalId: randomUUID(),
      workspaceId: h.resources.userWorkspace.id,
      environment: i % 2 === 0 ? environment : undefined,
    });
  }

  const root = await h.createRootKey(["identity.*.read_identity"]);

  const res = await h.get<V1IdentitiesListIdentitiesResponse>({
    url: `/v1/identities.listIdentities?environment=${environment}`,
    headers: {
      Authorization: `Bearer ${root.key}`,
    },
  });

  expect(res.status, `expected 200, received: ${JSON.stringify(res, null, 2)}`).toBe(200);
  expect(res.body.total).toBeGreaterThanOrEqual(5);
  expect(res.body.identities).toHaveLength(5);
});

test("returns ratelimits", async (t) => {
  const h = await IntegrationHarness.init(t);

  const identityId = newId("test");

  await h.db.primary.insert(schema.identities).values({
    id: identityId,
    externalId: randomUUID(),
    workspaceId: h.resources.userWorkspace.id,
  });

  const ratelimit = {
    id: newId("test"),
    name: randomUUID(),
    limit: 10,
    duration: 1000,
    identityId,

    workspaceId: h.resources.userWorkspace.id,
  };
  await h.db.primary.insert(schema.ratelimits).values(ratelimit);
  const root = await h.createRootKey(["identity.*.read_identity"]);

  const res = await h.get<V1IdentitiesListIdentitiesResponse>({
    url: "/v1/identities.listIdentities",
    headers: {
      Authorization: `Bearer ${root.key}`,
    },
  });

  expect(res.status, `expected 200, received: ${JSON.stringify(res, null, 2)}`).toBe(200);
  expect(res.body.identities).toHaveLength(1);
  const found = res.body.identities[0];
  expect(found.ratelimits.length).toBe(1);
  expect(found.ratelimits[0].name).toEqual(ratelimit.name);
  expect(found.ratelimits[0].limit).toEqual(ratelimit.limit);
  expect(found.ratelimits[0].duration).toEqual(ratelimit.duration);
});

test("with limit", async (t) => {
  const h = await IntegrationHarness.init(t);
  const identityIds = new Array(10).fill(0).map(() => newId("test"));
  for (let i = 0; i < identityIds.length; i++) {
    await h.db.primary.insert(schema.identities).values({
      id: identityIds[i],
      externalId: randomUUID(),
      workspaceId: h.resources.userWorkspace.id,
    });
  }

  const root = await h.createRootKey(["identity.*.read_identity"]);

  const res = await h.get<V1IdentitiesListIdentitiesResponse>({
    url: "/v1/identities.listIdentities?limit=2",
    headers: {
      Authorization: `Bearer ${root.key}`,
    },
  });
  expect(res.status, `expected 200, received: ${JSON.stringify(res, null, 2)}`).toBe(200);
  expect(res.body.total).toBeGreaterThanOrEqual(identityIds.length);
  expect(res.body.identities).toHaveLength(2);
}, 10_000);

test("with cursor", async (t) => {
  const h = await IntegrationHarness.init(t);
  const identityIds = new Array(10).fill(0).map(() => newId("test"));
  for (let i = 0; i < identityIds.length; i++) {
    await h.db.primary.insert(schema.identities).values({
      id: identityIds[i],
      externalId: randomUUID(),
      workspaceId: h.resources.userWorkspace.id,
    });
  }

  const root = await h.createRootKey(["identity.*.read_identity"]);
  const res1 = await h.get<V1IdentitiesListIdentitiesResponse>({
    url: "/v1/identities.listIdentities?limit=2",
    headers: {
      Authorization: `Bearer ${root.key}`,
    },
  });
  expect(res1.status).toEqual(200);
  expect(res1.body.cursor).toBeDefined();

  const res2 = await h.get<V1IdentitiesListIdentitiesResponse>({
    url: `/v1/identities.listIdentities?limit=3&cursor=${res1.body.cursor}`,
    headers: {
      Authorization: `Bearer ${root.key}`,
    },
  });

  expect(res2.status).toEqual(200);
  const found = new Set<string>();
  for (const key of res1.body.identities) {
    found.add(key.id);
  }
  for (const key of res2.body.identities) {
    found.add(key.id);
  }
  expect(found.size).toEqual(5);
});
