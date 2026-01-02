import { expect, test } from "vitest";

import { schema } from "@unkey/db";
import { newId } from "@unkey/id";
import { IntegrationHarness } from "src/pkg/testutil/integration-harness";

import { randomUUID } from "node:crypto";
import type { V1IdentitiesGetIdentityResponse } from "./v1_identities_getIdentity";

test("return the identity", async (t) => {
  const h = await IntegrationHarness.init(t);
  const root = await h.createRootKey(["identity.*.read_identity"]);

  const identity = {
    id: newId("test"),
    externalId: randomUUID(),
    workspaceId: h.resources.userWorkspace.id,
  };
  await h.db.primary.insert(schema.identities).values(identity);

  const res = await h.get<V1IdentitiesGetIdentityResponse>({
    url: `/v1/identities.getIdentity?identityId=${identity.id}`,
    headers: {
      Authorization: `Bearer ${root.key}`,
    },
  });

  expect(res.status, `expected 200, received: ${JSON.stringify(res, null, 2)}`).toBe(200);
  expect(res.body).toMatchObject({
    id: identity.id,
    externalId: identity.externalId,
  });
});

test("find the identity by its externalId", async (t) => {
  const h = await IntegrationHarness.init(t);
  const root = await h.createRootKey(["identity.*.read_identity"]);

  const identity = {
    id: newId("test"),
    externalId: randomUUID(),
    workspaceId: h.resources.userWorkspace.id,
  };
  await h.db.primary.insert(schema.identities).values(identity);

  const res = await h.get<V1IdentitiesGetIdentityResponse>({
    url: `/v1/identities.getIdentity?externalId=${identity.externalId}`,
    headers: {
      Authorization: `Bearer ${root.key}`,
    },
  });

  expect(res.status, `expected 200, received: ${JSON.stringify(res, null, 2)}`).toBe(200);
  expect(res.body).toMatchObject({
    id: identity.id,
    externalId: identity.externalId,
  });
});

test("with ratelimits", async (t) => {
  const h = await IntegrationHarness.init(t);
  const identity = {
    id: newId("test"),
    externalId: randomUUID(),
    workspaceId: h.resources.userWorkspace.id,
  };

  await h.db.primary.insert(schema.identities).values(identity);

  const ratelimit = {
    id: newId("test"),
    name: randomUUID(),
    limit: 10,
    duration: 100,
    workspaceId: h.resources.userWorkspace.id,
    identityId: identity.id,
  };
  await h.db.primary.insert(schema.ratelimits).values(ratelimit);

  const root = await h.createRootKey(["identity.*.read_identity"]);

  const res = await h.get<V1IdentitiesGetIdentityResponse>({
    url: `/v1/identities.getIdentity?identityId=${identity.id}`,
    headers: {
      Authorization: `Bearer ${root.key}`,
    },
  });

  expect(res.status, `expected 200, received: ${JSON.stringify(res, null, 2)}`).toBe(200);
  expect(res.body).toMatchObject({
    id: identity.id,
    externalId: identity.externalId,
  });

  expect(res.body.ratelimits.length).toBe(1);
  expect(res.body.ratelimits[0].name).toBe(ratelimit.name);
  expect(res.body.ratelimits[0].duration).toBe(ratelimit.duration);
  expect(res.body.ratelimits[0].limit).toBe(ratelimit.limit);
});
