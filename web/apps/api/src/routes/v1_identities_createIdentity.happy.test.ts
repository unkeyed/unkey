import { describe, expect, test } from "vitest";

import { randomUUID } from "node:crypto";
import { IntegrationHarness } from "src/pkg/testutil/integration-harness";

import type {
  V1IdentitiesCreateIdentityRequest,
  V1IdentitiesCreateIdentityResponse,
} from "./v1_identities_createIdentity";

test("creates new identity", async (t) => {
  const h = await IntegrationHarness.init(t);
  const root = await h.createRootKey(["identity.*.create_identity"]);
  const res = await h.post<V1IdentitiesCreateIdentityRequest, V1IdentitiesCreateIdentityResponse>({
    url: "/v1/identities.createIdentity",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${root.key}`,
    },
    body: {
      externalId: randomUUID(),
    },
  });

  expect(res.status, `expected 200, received: ${JSON.stringify(res, null, 2)}`).toBe(200);

  const found = await h.db.primary.query.identities.findFirst({
    where: (table, { eq }) => eq(table.id, res.body.identityId),
  });
  expect(found).toBeDefined();
});

describe("with meta", () => {
  test("stores metadata", async (t) => {
    const h = await IntegrationHarness.init(t);
    const root = await h.createRootKey(["identity.*.create_identity"]);
    const res = await h.post<V1IdentitiesCreateIdentityRequest, V1IdentitiesCreateIdentityResponse>(
      {
        url: "/v1/identities.createIdentity",
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${root.key}`,
        },
        body: {
          externalId: randomUUID(),
          meta: { hello: "world" },
        },
      },
    );

    expect(res.status, `expected 200, received: ${JSON.stringify(res, null, 2)}`).toBe(200);

    const found = await h.db.primary.query.identities.findFirst({
      where: (table, { eq }) => eq(table.id, res.body.identityId),
    });
    expect(found).toBeDefined();
    expect(found!.meta).toMatchObject({ hello: "world" });
  });
});

describe("with ratelimits", () => {
  test("stores ratelimits", async (t) => {
    const h = await IntegrationHarness.init(t);
    const root = await h.createRootKey(["identity.*.create_identity"]);

    const ratelimits = [
      {
        name: "tokens",
        limit: 10,
        duration: 1000,
      },
      {
        name: "requests",
        limit: 1,
        duration: 60000,
      },
    ];

    const res = await h.post<V1IdentitiesCreateIdentityRequest, V1IdentitiesCreateIdentityResponse>(
      {
        url: "/v1/identities.createIdentity",
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${root.key}`,
        },
        body: {
          externalId: randomUUID(),
          ratelimits,
        },
      },
    );

    expect(res.status, `expected 200, received: ${JSON.stringify(res, null, 2)}`).toBe(200);

    const found = await h.db.primary.query.identities.findFirst({
      where: (table, { eq }) => eq(table.id, res.body.identityId),
      with: {
        ratelimits: true,
      },
    });
    expect(found).toBeDefined();
    for (const rl of ratelimits) {
      expect(
        found!.ratelimits.some(
          (f) =>
            f.identityId === res.body.identityId &&
            f.name === rl.name &&
            f.limit === rl.limit &&
            f.duration === rl.duration,
        ),
      ).toBe(true);
    }
  });
});
