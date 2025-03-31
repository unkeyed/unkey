import { describe, expect, test } from "vitest";

import { randomUUID } from "node:crypto";
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

test("creating the same role twice errors", async (t) => {
  const h = await IntegrationHarness.init(t);
  const root = await h.createRootKey(["rbac.*.create_role"]);

  const createRoleRequest = {
    url: "/v1/permissions.createRole",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${root.key}`,
    },
    body: {
      name: randomUUID(),
    },
  };

  const successResponse = await h.post<
    V1PermissionsCreateRoleRequest,
    V1PermissionsCreateRoleResponse
  >(createRoleRequest);

  expect(
    successResponse.status,
    `expected 200, received: ${JSON.stringify(successResponse, null, 2)}`,
  ).toBe(200);

  const errorResponse = await h.post<
    V1PermissionsCreateRoleRequest,
    V1PermissionsCreateRoleResponse
  >(createRoleRequest);

  expect(
    errorResponse.status,
    `expected 409, received: ${JSON.stringify(errorResponse, null, 2)}`,
  ).toBe(409);
  expect(errorResponse.body).toMatchObject({
    error: {
      code: "CONFLICT",
      docs: "https://unkey.dev/docs/api-reference/errors/code/CONFLICT",
      message: `Role with name "${createRoleRequest.body.name}" already exists in this workspace`,
    },
  });
});
