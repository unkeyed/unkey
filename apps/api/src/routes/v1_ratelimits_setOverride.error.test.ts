import { expect, test } from "vitest";

import { randomUUID } from "node:crypto";
import { IntegrationHarness } from "src/pkg/testutil/integration-harness";

import { schema } from "@unkey/db";
import { newId } from "@unkey/id";
import type {
  V1RatelimitSetOverrideRequest,
  V1RatelimitSetOverrideResponse,
} from "./v1_ratelimits_setOverride";

test("Missing Namespace", async (t) => {
  const h = await IntegrationHarness.init(t);
  const root = await h.createRootKey([
    "*",
    "ratelimit.*.set_Override",
    "ratelimit.*.create_namespace",
    "ratelimit.*.read_override",
  ]);
  const identifier = randomUUID();
  const namespaceId = newId("test");

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
  expect(res.status, `expected 404, received: ${JSON.stringify(res, null, 2)}`).toBe(404);
  expect(res.body).toMatchObject({
    error: {
      code: "NOT_FOUND",
      docs: "https://unkey.dev/docs/api-reference/errors/code/NOT_FOUND",
      message: `Namespace ${namespaceId} not found`,
    },
  });
});
test("Empty Identifier string", async (t) => {
  const h = await IntegrationHarness.init(t);
  const root = await h.createRootKey([
    "*",
    "ratelimit.*.set_Override",
    "ratelimit.*.create_namespace",
    "ratelimit.*.read_override",
  ]);

  const namespaceId = newId("test");

  const namespace = {
    id: namespaceId,
    workspaceId: h.resources.userWorkspace.id,
    name: randomUUID(),
    createdAtM: Date.now(),
  };

  await h.db.primary.insert(schema.ratelimitNamespaces).values(namespace);

  const override = {
    namespaceId: namespaceId,
    identifier: "",
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
  expect(res.status, `expected 400, received: ${JSON.stringify(res, null, 2)}`).toBe(400);
  expect(res.body).toMatchObject({
    error: {
      code: "BAD_REQUEST",
      docs: "https://unkey.dev/docs/api-reference/errors/code/BAD_REQUEST",
      message: "identifier: String must contain at least 3 character(s)",
    },
  });
});
