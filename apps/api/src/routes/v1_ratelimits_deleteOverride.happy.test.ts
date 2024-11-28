import { expect, test } from "vitest";

import { randomUUID } from "node:crypto";
import { IntegrationHarness } from "src/pkg/testutil/integration-harness";

import { isNull, schema } from "@unkey/db";
import { newId } from "@unkey/id";
import type {
  V1RatelimitDeleteOverrideRequest,
  V1RatelimitDeleteOverrideResponse,
} from "./v1_ratelimits_deleteOverride";

test("deletes override", async (t) => {
  const h = await IntegrationHarness.init(t);

  const overrideId = newId("test");
  const identifier = randomUUID();
  const namespaceId = newId("test");
  const namespace = {
    id: namespaceId,
    workspaceId: h.resources.userWorkspace.id,
    createdAt: new Date(),
    name: newId("test"),
  };
  await h.db.primary.insert(schema.ratelimitNamespaces).values(namespace);

  await h.db.primary.insert(schema.ratelimitOverrides).values({
    id: overrideId,
    workspaceId: h.resources.userWorkspace.id,
    namespaceId,
    identifier,
    limit: 1,
    duration: 60_000,
    async: false,
  });

  const root = await h.createRootKey(["ratelimit.*.delete_override"]);
  const res = await h.post<V1RatelimitDeleteOverrideRequest, V1RatelimitDeleteOverrideResponse>({
    url: "/v1/ratelimits.deleteOverride",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${root.key}`,
    },
    body: {
      namespaceId,
      identifier,
    },
  });

  expect(res.status, `expected 200, received: ${JSON.stringify(res, null, 2)}`).toBe(200);

  const found = await h.db.primary.query.ratelimitOverrides.findFirst({
    where: (table, { eq, and }) => and(eq(table.id, overrideId), isNull(table.deletedAt)),
  });
  expect(found).toBeUndefined();
});
