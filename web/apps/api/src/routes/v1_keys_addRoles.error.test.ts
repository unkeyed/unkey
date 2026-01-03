import { expect, test } from "vitest";

import { IntegrationHarness } from "src/pkg/testutil/integration-harness";

import type { V1KeysAddRolesRequest, V1KeysAddRolesResponse } from "./v1_keys_addRoles";

test("empty keyId", async (t) => {
  const h = await IntegrationHarness.init(t);
  const { key: rootKey } = await h.createRootKey(["*"]);

  const res = await h.post<V1KeysAddRolesRequest, V1KeysAddRolesResponse>({
    url: "/v1/keys.addRoles",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${rootKey}`,
    },
    body: {
      keyId: "",
      roles: [{ id: "role123" }],
    },
  });

  expect(res.status).toEqual(400);
  expect(res.body).toMatchObject({
    error: {
      code: "BAD_REQUEST",
      docs: "https://unkey.dev/docs/api-reference/errors/code/BAD_REQUEST",
      message: "keyId: String must contain at least 1 character(s)",
    },
  });
});

test("empty roles", async (t) => {
  const h = await IntegrationHarness.init(t);
  const { key: rootKey } = await h.createRootKey(["*"]);

  const res = await h.post<V1KeysAddRolesRequest, V1KeysAddRolesResponse>({
    url: "/v1/keys.addRoles",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${rootKey}`,
    },
    body: {
      keyId: "key_123",
      roles: [],
    },
  });

  expect(res.status).toEqual(400);
  expect(res.body).toMatchObject({
    error: {
      code: "BAD_REQUEST",
      docs: "https://unkey.dev/docs/api-reference/errors/code/BAD_REQUEST",
      message: "roles: Array must contain at least 1 element(s)",
    },
  });
});
