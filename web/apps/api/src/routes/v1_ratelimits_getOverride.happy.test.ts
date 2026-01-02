import { randomUUID } from "node:crypto";
import { schema } from "@unkey/db";
import { newId } from "@unkey/id";
import { IntegrationHarness } from "src/pkg/testutil/integration-harness";
import { expect, test } from "vitest";
import type { V1RatelimitGetOverrideResponse } from "./v1_ratelimits_getOverride";

test("return a single override using namespaceId", async (t) => {
  const h = await IntegrationHarness.init(t);
  const root = await h.createRootKey(["ratelimit.*.read_override"]);
  const namespaceId = newId("test");
  const namespaceName = randomUUID();
  const overrideId = newId("test");
  const identifier = randomUUID();

  // Namespace
  const namespace = {
    id: namespaceId,
    name: namespaceName,
    workspaceId: h.resources.userWorkspace.id,
    createdAtM: Date.now(),
  };
  await h.db.primary.insert(schema.ratelimitNamespaces).values(namespace);
  // Initial Override
  await h.db.primary.insert(schema.ratelimitOverrides).values({
    id: overrideId,
    workspaceId: h.resources.userWorkspace.id,
    namespaceId: namespaceId,
    identifier: identifier,
    limit: 1,
    duration: 60_000,
    async: false,
  });

  const res = await h.get<V1RatelimitGetOverrideResponse>({
    url: `/v1/ratelimits.getOverride?namespaceId=${namespaceId}&identifier=${identifier}`,
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${root.key}`,
    },
  });
  expect(res.status, `expected 200, received: ${JSON.stringify(res, null, 2)}`).toBe(200);
  expect(res.body.id).toBe(overrideId);
  expect(res.body.identifier).toEqual(identifier);
  expect(res.body.limit).toEqual(1);
  expect(res.body.duration).toEqual(60_000);
  expect(res.body.async).toEqual(false);
});

test("return a single override using namespaceName", async (t) => {
  const h = await IntegrationHarness.init(t);
  const root = await h.createRootKey(["ratelimit.*.read_override"]);
  const namespaceId = newId("test");
  const namespaceName = randomUUID();
  const overrideId = newId("test");
  const identifier = randomUUID();

  // Namespace
  const namespace = {
    id: namespaceId,
    name: namespaceName,
    workspaceId: h.resources.userWorkspace.id,
    createdAtM: Date.now(),
  };
  await h.db.primary.insert(schema.ratelimitNamespaces).values(namespace);
  await h.db.primary.insert(schema.ratelimitOverrides).values({
    id: overrideId,
    workspaceId: h.resources.userWorkspace.id,
    namespaceId: namespaceId,
    identifier: identifier,
    limit: 1,
    duration: 60_000,
    async: false,
  });

  const res = await h.get<V1RatelimitGetOverrideResponse>({
    url: `/v1/ratelimits.getOverride?namespaceName=${namespaceName}&identifier=${identifier}`,
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${root.key}`,
    },
  });
  expect(res.status, `expected 200, received: ${JSON.stringify(res, null, 2)}`).toBe(200);
  expect(res.body.id).toBe(overrideId);
  expect(res.body.identifier).toEqual(identifier);
  expect(res.body.limit).toEqual(1);
  expect(res.body.duration).toEqual(60_000);
  expect(res.body.async).toEqual(false);
});
