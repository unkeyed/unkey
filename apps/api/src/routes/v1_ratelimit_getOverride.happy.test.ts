import { randomUUID } from "node:crypto";
import { schema } from "@unkey/db";
import { newId } from "@unkey/id";
import { IntegrationHarness } from "src/pkg/testutil/integration-harness";
import { expect, test } from "vitest";
import type { V1RatelimitGetOverrideResponse } from "./v1_ratelimit_getOverride";

test("return a single override", async (t) => {
  const h = await IntegrationHarness.init(t);
  const root = await h.createRootKey(["ratelimit.*.read_override"]);
  const namespaceId = newId("test");
  const namespaceName = "Test.Name";
  const overrideId = newId("test");
  const identifier = randomUUID();

  // Namespace
  const namespace = {
    id: namespaceId,
    name: namespaceName,
    workspaceId: h.resources.userWorkspace.id,
    createdAt: new Date(),
  };
  const namespaceRes = await h.db.primary.insert(schema.ratelimitNamespaces).values(namespace);
  // Initial Override
  console.log(namespaceRes);

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
    url: `/v1/ratelimit.getOverride?namespaceId=${namespaceId}&identifier=${identifier}`,
    headers: {
      Authorization: `Bearer ${root.key}`,
    },
  });
  console.log(res);

  expect(res.status, `expected 200, received: ${JSON.stringify(res, null, 2)}`).toBe(200);
  expect(res.body.id).toBe(overrideId);
  expect(res.body.identifier).toEqual(identifier);
  expect(res.body.limit).toEqual(1);
  expect(res.body.duration).toEqual(60_000);
  expect(res.body.async).toEqual(false);
});
