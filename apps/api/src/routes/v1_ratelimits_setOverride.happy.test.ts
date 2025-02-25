import { expect, test } from "vitest";

import { randomUUID } from "node:crypto";
import { IntegrationHarness } from "src/pkg/testutil/integration-harness";

import { schema } from "@unkey/db";
import { newId } from "@unkey/id";
import type {
  V1RatelimitSetOverrideRequest,
  V1RatelimitSetOverrideResponse,
} from "./v1_ratelimits_setOverride";

test("Set ratelimit override", async (t) => {
  const h = await IntegrationHarness.init(t);
  const root = await h.createRootKey([
    "*",
    "ratelimit.*.set_override",
    "ratelimit.*.create_namespace",
    "ratelimit.*.read_override",
  ]);
  const identifier = randomUUID();
  const namespaceId = newId("test");

  const namespace = {
    id: namespaceId,
    workspaceId: h.resources.userWorkspace.id,
    name: randomUUID(),
    createdAtM: Date.now(),
  };

  await h.db.primary.insert(schema.ratelimitNamespaces).values(namespace);

  await h.db.primary.query.ratelimitNamespaces.findFirst({
    where: (table, { eq }) => eq(table.id, namespaceId),
  });

  const override = {
    namespaceId: namespaceId,
    identifier: identifier,
    limit: 10,
    duration: 6500,
    async: true,
  };
  const res = await h.post<V1RatelimitSetOverrideRequest, V1RatelimitSetOverrideResponse>({
    url: "/v1/ratelimits.setOverride",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${root.key}`,
    },
    body: override,
  });

  expect(res.status, `expected 200, received: ${JSON.stringify(res, null, 2)}`).toBe(200);

  const resInit = await h.db.primary.query.ratelimitOverrides.findFirst({
    where: (table, { eq, and }) =>
      and(eq(table.namespaceId, namespaceId), eq(table.identifier, identifier)),
  });
  expect(resInit).toBeDefined();

  const resUpdate = await h.post<V1RatelimitSetOverrideRequest, V1RatelimitSetOverrideResponse>({
    url: "/v1/ratelimits.setOverride",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${root.key}`,
    },
    body: {
      namespaceId: resInit?.namespaceId,
      identifier: resInit?.identifier!,
      limit: 10,
      duration: 50000,
      async: true,
    },
  });

  expect(resUpdate.status, `expected 200, received: ${JSON.stringify(resUpdate, null, 2)}`).toBe(
    200,
  );

  const resNew = await h.db.primary.query.ratelimitOverrides.findFirst({
    where: (table, { eq, and }) =>
      and(eq(table.namespaceId, namespaceId), eq(table.identifier, identifier)),
  });

  expect(resNew?.identifier).toEqual(identifier);
  expect(resNew?.namespaceId).toEqual(namespaceId);
  expect(resNew?.limit).toEqual(10);
  expect(resNew?.duration).toEqual(50000);
  expect(resNew?.async).toEqual(true);
});
