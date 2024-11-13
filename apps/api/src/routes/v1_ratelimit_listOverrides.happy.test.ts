import { randomUUID } from "node:crypto";
import { schema } from "@unkey/db";
import { newId } from "@unkey/id";
import { IntegrationHarness } from "src/pkg/testutil/integration-harness";
import { expect, test } from "vitest";
import type { V1RatelimitListOverridesResponse } from "./v1_ratelimit_listOverrides";

test("return all overrides", async (t) => {
  const h = await IntegrationHarness.init(t);
  const root = await h.createRootKey(["ratelimit.*.read_override"]);
  const namespaceId = newId("test");
  const namespaceName = "test.Name";
  const overrideId = newId("test");
  const identifier = randomUUID();

  // Namespace
  const namespace = {
    id: namespaceId,
    name: namespaceName,
    workspaceId: h.resources.userWorkspace.id,
    createdAt: new Date(),
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

  const res = await h.get<V1RatelimitListOverridesResponse>({
    url: `/v1/ratelimit.listOverrides?namespaceId=${namespaceId}&identifier=${identifier}`,
    headers: {
      Authorization: `Bearer ${root.key}`,
    },
  });

  expect(res.status, `expected 200, received: ${JSON.stringify(res, null, 2)}`).toBe(200);
  expect(res.body.total).toBe(1);
  expect(res.body.overrides[0].id).toEqual(overrideId);
  expect(res.body.overrides[0].identifier).toEqual(identifier);
});
