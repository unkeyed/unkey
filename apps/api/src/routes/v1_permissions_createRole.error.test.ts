import { describe, expect, test } from "vitest";

import { IntegrationHarness } from "src/pkg/testutil/integration-harness";

import type {
  V1PermissionsCreateRoleRequest,
  V1PermissionsCreateRoleResponse,
} from "./v1_permissions_createRole";

describe.each([
  { name: "empty name", roleName: "" },
  { name: "short name", roleName: "ab" },
])("$name", ({ roleName }) => {
  test("reject", async (t) => {
    const h = await IntegrationHarness.init(t);
    const { key: rootKey } = await h.createRootKey(["*"]);

    const res = await h.post<V1PermissionsCreateRoleRequest, V1PermissionsCreateRoleResponse>({
      url: "/v1/permissions.createRole",
      headers: {
        "Content-Type": "application/json",
        Authorization: `Bearer ${rootKey}`,
      },
      body: {
        name: roleName,
      },
    });

    expect(res.status).toEqual(400);
    expect(res.body).toMatchObject({
      error: {
        code: "BAD_REQUEST",
        docs: "https://unkey.dev/docs/api-reference/errors/code/BAD_REQUEST",
        message: "name: String must contain at least 3 character(s)",
      },
    });
  });
});
