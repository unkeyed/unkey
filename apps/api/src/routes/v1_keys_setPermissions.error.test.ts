import { expect, test } from "vitest";

import { IntegrationHarness } from "src/pkg/testutil/integration-harness";

import type {
  V1KeysSetPermissionsRequest,
  V1KeysSetPermissionsResponse,
} from "./v1_keys_setPermissions";

test("empty keyId", async (t) => {
  const h = await IntegrationHarness.init(t);
  const { key: rootKey } = await h.createRootKey(["*"]);

  const res = await h.post<V1KeysSetPermissionsRequest, V1KeysSetPermissionsResponse>({
    url: "/v1/keys.setPermissions",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${rootKey}`,
    },
    body: {
      keyId: "",
      permissions: [{ id: "perm_123" }],
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

test("empty permissions", async (t) => {
  const h = await IntegrationHarness.init(t);
  const { key: rootKey } = await h.createRootKey(["*"]);

  const res = await h.post<V1KeysSetPermissionsRequest, V1KeysSetPermissionsResponse>({
    url: "/v1/keys.setPermissions",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${rootKey}`,
    },
    body: {
      keyId: "key_123",
      permissions: [],
    },
  });

  expect(res.status).toEqual(400);
  expect(res.body).toMatchObject({
    error: {
      code: "BAD_REQUEST",
      docs: "https://unkey.dev/docs/api-reference/errors/code/BAD_REQUEST",
      message: "permissions: Array must contain at least 1 element(s)",
    },
  });
});
