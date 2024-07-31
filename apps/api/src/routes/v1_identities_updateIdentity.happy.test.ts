import { expect, test } from "vitest";

import { randomUUID } from "node:crypto";
import { IntegrationHarness } from "src/pkg/testutil/integration-harness";

import { schema } from "@unkey/db";
import { newId } from "@unkey/id";
import type {
  V1IdentitiesUpdateIdentityRequest,
  V1IdentitiesUpdateIdentityResponse,
} from "./v1_identities_updateIdentity";

test("can be identified via externalId", async (t) => {
  const h = await IntegrationHarness.init(t);
  const root = await h.createRootKey(["identity.*.update_identity"]);

  const identity = {
    id: newId("test"),
    workspaceId: h.resources.userWorkspace.id,
    externalId: randomUUID(),
    environment: randomUUID(),
  };

  await h.db.primary.insert(schema.identities).values(identity);

  await h.db.primary.insert(schema.ratelimits).values({
    id: newId("test"),
    identityId: identity.id,
    name: randomUUID(),
    limit: 123123,
    duration: 131251,
    workspaceId: h.resources.userWorkspace.id,
  });

  const ratelimits = [
    {
      name: randomUUID(),
      limit: 10,
      duration: 1000,
    },
    {
      name: randomUUID(),
      limit: 1000,
      duration: 11111111,
    },
  ];

  const res = await h.post<V1IdentitiesUpdateIdentityRequest, V1IdentitiesUpdateIdentityResponse>({
    url: "/v1/identities.updateIdentity",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${root.key}`,
    },
    body: {
      externalId: identity.externalId,
      environment: identity.environment,
      ratelimits,
    },
  });

  expect(res.status, `expected 200, received: ${JSON.stringify(res, null, 2)}`).toBe(200);

  const found = await h.db.primary.query.ratelimits.findMany({
    where: (table, { eq }) => eq(table.identityId, identity.id),
  });

  expect(found.length).toBe(ratelimits.length);
  for (const rl of ratelimits) {
    expect(
      found.some((f) => f.name === rl.name && f.limit === rl.limit && f.duration === rl.duration),
    );
  }
});

test("sets new ratelimits", async (t) => {
  const h = await IntegrationHarness.init(t);
  const root = await h.createRootKey(["identity.*.update_identity"]);

  const identity = {
    id: newId("test"),
    workspaceId: h.resources.userWorkspace.id,
    externalId: randomUUID(),
  };

  await h.db.primary.insert(schema.identities).values(identity);

  await h.db.primary.insert(schema.ratelimits).values({
    id: newId("test"),
    identityId: identity.id,
    name: randomUUID(),
    limit: 123123,
    duration: 131251,
    workspaceId: h.resources.userWorkspace.id,
  });

  const ratelimits = [
    {
      name: randomUUID(),
      limit: 10,
      duration: 1000,
    },
    {
      name: randomUUID(),
      limit: 1000,
      duration: 11111111,
    },
  ];

  const res = await h.post<V1IdentitiesUpdateIdentityRequest, V1IdentitiesUpdateIdentityResponse>({
    url: "/v1/identities.updateIdentity",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${root.key}`,
    },
    body: {
      identityId: identity.id,
      ratelimits,
    },
  });

  expect(res.status, `expected 200, received: ${JSON.stringify(res, null, 2)}`).toBe(200);

  const found = await h.db.primary.query.ratelimits.findMany({
    where: (table, { eq }) => eq(table.identityId, identity.id),
  });

  expect(found.length).toBe(ratelimits.length);
  for (const rl of ratelimits) {
    expect(
      found.some((f) => f.name === rl.name && f.limit === rl.limit && f.duration === rl.duration),
    );
  }
});

test("works with hundreds of keys", async (t) => {
  const h = await IntegrationHarness.init(t);
  const root = await h.createRootKey(["identity.*.update_identity"]);

  const identity = {
    id: newId("test"),
    workspaceId: h.resources.userWorkspace.id,
    externalId: randomUUID(),
  };

  await h.db.primary.insert(schema.identities).values(identity);

  for (let i = 0; i < 200; i++) {
    await h.createKey({ identityId: identity.id });
  }

  await h.db.primary.insert(schema.ratelimits).values({
    id: newId("test"),
    identityId: identity.id,
    name: randomUUID(),
    limit: 123123,
    duration: 131251,
    workspaceId: h.resources.userWorkspace.id,
  });

  const ratelimits = [
    {
      name: randomUUID(),
      limit: 10,
      duration: 1000,
    },
    {
      name: randomUUID(),
      limit: 1000,
      duration: 11111111,
    },
  ];

  const res = await h.post<V1IdentitiesUpdateIdentityRequest, V1IdentitiesUpdateIdentityResponse>({
    url: "/v1/identities.updateIdentity",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${root.key}`,
    },
    body: {
      identityId: identity.id,
      ratelimits,
    },
  });

  expect(res.status, `expected 200, received: ${JSON.stringify(res, null, 2)}`).toBe(200);

  const found = await h.db.primary.query.ratelimits.findMany({
    where: (table, { eq }) => eq(table.identityId, identity.id),
  });

  expect(found.length).toBe(ratelimits.length);
  for (const rl of ratelimits) {
    expect(
      found.some((f) => f.name === rl.name && f.limit === rl.limit && f.duration === rl.duration),
    );
  }
});

test("updates meta", async (t) => {
  const h = await IntegrationHarness.init(t);
  const root = await h.createRootKey(["identity.*.update_identity"]);

  const identity = {
    id: newId("test"),
    workspaceId: h.resources.userWorkspace.id,
    externalId: randomUUID(),
  };

  await h.db.primary.insert(schema.identities).values(identity);

  const res = await h.post<V1IdentitiesUpdateIdentityRequest, V1IdentitiesUpdateIdentityResponse>({
    url: "/v1/identities.updateIdentity",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${root.key}`,
    },
    body: {
      identityId: identity.id,
      meta: {
        hello: "world",
      },
    },
  });

  expect(res.status, `expected 200, received: ${JSON.stringify(res, null, 2)}`).toBe(200);

  const found = await h.db.primary.query.identities.findFirst({
    where: (table, { eq }) => eq(table.id, identity.id),
  });

  expect(found).toBeDefined();
  expect(found!.meta).toMatchObject({
    hello: "world",
  });
});

test("updating meta does not affect ratelimits", async (t) => {
  const h = await IntegrationHarness.init(t);
  const root = await h.createRootKey(["identity.*.update_identity"]);

  const identity = {
    id: newId("test"),
    workspaceId: h.resources.userWorkspace.id,
    externalId: randomUUID(),
  };

  await h.db.primary.insert(schema.identities).values(identity);

  const ratelimit = {
    id: newId("test"),
    identityId: identity.id,
    name: randomUUID(),
    limit: 123123,
    duration: 131251,
    workspaceId: h.resources.userWorkspace.id,
  };
  await h.db.primary.insert(schema.ratelimits).values(ratelimit);

  const res = await h.post<V1IdentitiesUpdateIdentityRequest, V1IdentitiesUpdateIdentityResponse>({
    url: "/v1/identities.updateIdentity",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${root.key}`,
    },
    body: {
      identityId: identity.id,
      meta: {
        x: 1,
      },
    },
  });

  expect(res.status, `expected 200, received: ${JSON.stringify(res, null, 2)}`).toBe(200);

  const found = await h.db.primary.query.identities.findFirst({
    where: (table, { eq }) => eq(table.id, identity.id),
    with: {
      ratelimits: true,
    },
  });

  expect(found?.ratelimits.length).toBe(1);
  expect(found?.ratelimits[0]).toMatchObject(ratelimit);
});
