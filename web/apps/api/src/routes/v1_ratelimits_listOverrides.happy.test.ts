import { randomUUID } from "node:crypto";
import { schema } from "@unkey/db";
import { newId } from "@unkey/id";
import { IntegrationHarness } from "src/pkg/testutil/integration-harness";
import { expect, test } from "vitest";
import type { V1RatelimitListOverridesResponse } from "./v1_ratelimits_listOverrides";

// Test case for Multiple Overrides for the Same Namespace
test("return multiple overrides for the same namespace", async (t) => {
  const h = await IntegrationHarness.init(t);
  const root = await h.createRootKey(["ratelimit.*.read_override"]);
  const namespaceId = newId("test");
  const namespaceName = randomUUID();

  // Insert namespace
  await h.db.primary.insert(schema.ratelimitNamespaces).values({
    id: namespaceId,
    name: namespaceName,
    workspaceId: h.resources.userWorkspace.id,
    createdAtM: Date.now(),
  });

  // Insert multiple overrides
  const overrides = [
    {
      id: newId("test"),
      workspaceId: h.resources.userWorkspace.id,
      namespaceId: namespaceId,
      identifier: randomUUID(),
      limit: 1,
      duration: 60_000,
      async: false,
    },
    {
      id: newId("test"),
      workspaceId: h.resources.userWorkspace.id,
      namespaceId: namespaceId,
      identifier: randomUUID(),
      limit: 2,
      duration: 120_000,
      async: true,
    },
  ];
  await h.db.primary.insert(schema.ratelimitOverrides).values(overrides);

  const res = await h.get<V1RatelimitListOverridesResponse>({
    url: `/v1/ratelimits.listOverrides?namespaceId=${namespaceId}`,
    headers: {
      Authorization: `Bearer ${root.key}`,
    },
  });

  expect(res.status).toBe(200);
  expect(res.body.total).toBe(2);
  expect(res.body.overrides.length).toBe(2);
});

// Test case for No Overrides Found
test("return empty list when no overrides exist", async (t) => {
  const h = await IntegrationHarness.init(t);
  const root = await h.createRootKey(["ratelimit.*.read_override"]);
  const namespaceId = newId("test");

  // Insert namespace without overrides
  await h.db.primary.insert(schema.ratelimitNamespaces).values({
    id: namespaceId,
    name: randomUUID(),
    workspaceId: h.resources.userWorkspace.id,
    createdAtM: Date.now(),
  });

  const res = await h.get<V1RatelimitListOverridesResponse>({
    url: `/v1/ratelimits.listOverrides?namespaceId=${namespaceId}`,
    headers: {
      Authorization: `Bearer ${root.key}`,
    },
  });

  expect(res.status).toBe(200);
  expect(res.body.total).toBe(0);
  expect(res.body.overrides.length).toBe(0);
});

// Test case for Invalid Identifier
test("return empty list when none exist", async (t) => {
  const h = await IntegrationHarness.init(t);
  const root = await h.createRootKey(["ratelimit.*.read_override"]);
  const namespaceId = newId("test");
  const invalidIdentifier = randomUUID();

  // Insert namespace
  await h.db.primary.insert(schema.ratelimitNamespaces).values({
    id: namespaceId,
    name: randomUUID(),
    workspaceId: h.resources.userWorkspace.id,
    createdAtM: Date.now(),
  });

  // Insert an override with a different identifier

  const res = await h.get<V1RatelimitListOverridesResponse>({
    url: `/v1/ratelimits.listOverrides?namespaceId=${namespaceId}&identifier=${invalidIdentifier}`,
    headers: {
      Authorization: `Bearer ${root.key}`,
    },
  });

  expect(res.status).toBe(200);
  expect(res.body.total).toBe(0);
  expect(res.body.overrides.length).toBe(0);
});
