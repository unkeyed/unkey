import { randomUUID } from "node:crypto";
import { schema } from "@unkey/db";
import { newId } from "@unkey/id";
import { IntegrationHarness } from "src/pkg/testutil/integration-harness";
import { expect, test } from "vitest";
import type { V1RatelimitGetOverrideResponse } from "./v1_ratelimits_getOverride";

test("Missing Namespace", async (t) => {
  const h = await IntegrationHarness.init(t);
  const root = await h.createRootKey(["ratelimit.*.read_override"]);
  const namespaceId = newId("test");
  const namespaceName = "Test.Name";
  const overrideId = newId("test");
  const identifier = randomUUID();

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
    url: `/v1/ratelimits.getOverride?namespaceId=&identifier=${identifier}`,
    headers: {
      Authorization: `Bearer ${root.key}`,
    },
  });
  expect(res.status, `expected 400, received: ${JSON.stringify(res, null, 2)}`).toBe(400);
  expect(res.body).toMatchObject({
    error: {
      code: "BAD_REQUEST",
      docs: "https://unkey.dev/docs/api-reference/errors/code/BAD_REQUEST",
      message: "You must provide a namespaceId or a namespaceName",
    },
  });
});
