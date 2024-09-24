import { expect, test } from "vitest";

import { randomInt, randomUUID } from "node:crypto";
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

  await new Promise((r) => setTimeout(r, 2000));

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

test("sets the same ratelimits again", async (t) => {
  const h = await IntegrationHarness.init(t);
  const root = await h.createRootKey(["identity.*.update_identity"]);

  const identity = {
    id: newId("test"),
    workspaceId: h.resources.userWorkspace.id,
    externalId: randomUUID(),
  };

  await h.db.primary.insert(schema.identities).values(identity);

  const ratelimits = new Array(6).fill(null).map((_, i) => ({
    name: randomUUID(),
    limit: 10 * (i + 1),
    duration: 1000 * (i + 1),
  }));

  for (let i = 0; i < 10; i++) {
    const res = await h.post<V1IdentitiesUpdateIdentityRequest, V1IdentitiesUpdateIdentityResponse>(
      {
        url: "/v1/identities.updateIdentity",
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${root.key}`,
        },
        body: {
          identityId: identity.id,
          ratelimits: ratelimits,
        },
      },
    );

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
  }
});

test("works with thousands of keys", { timeout: 300_000 }, async (t) => {
  const h = await IntegrationHarness.init(t);
  const root = await h.createRootKey(["identity.*.update_identity"]);

  const identity = {
    id: newId("test"),
    workspaceId: h.resources.userWorkspace.id,
    externalId: randomUUID(),
  };

  await h.db.primary.insert(schema.identities).values(identity);

  const keyIds: string[] = [];
  for (let i = 0; i < 1000; i++) {
    const key = await h.createKey({ identityId: identity.id });
    keyIds.push(key.keyId);
  }

  await h.db.primary.insert(schema.ratelimits).values({
    id: newId("test"),
    identityId: identity.id,
    name: randomUUID(),
    limit: 123123,
    duration: 131251,
    workspaceId: h.resources.userWorkspace.id,
  });

  const ratelimits = new Array(20).fill(null).map((_) => ({
    name: randomUUID(),
    limit: 10 + randomInt(100),
    duration: (1 + randomInt(1000)) * 1000,
  }));

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

  for (const keyId of keyIds) {
    const key = await h.db.primary.query.keys.findFirst({
      where: (table, { eq }) => eq(table.id, keyId),
      with: {
        identity: {
          with: {
            ratelimits: true,
          },
        },
      },
    });
    expect(key).toBeDefined();
    expect(key!.identity).toBeDefined();
    expect(key!.identity!.ratelimits.length).toBe(ratelimits.length);
    for (const rl of ratelimits) {
      expect(
        key!.identity!.ratelimits.some(
          (r) => r.name === rl.name && r.limit === rl.limit && r.duration === rl.duration,
        ),
      );
    }
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
