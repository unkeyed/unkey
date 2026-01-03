import { expect, test } from "vitest";

import { newId } from "@unkey/id";
import { IntegrationHarness } from "src/pkg/testutil/integration-harness";

import type { V1PermissionsGetRoleResponse } from "./v1_permissions_getRole";

test("role does not exist", async (t) => {
  const h = await IntegrationHarness.init(t);
  const roleId = newId("test");

  const root = await h.createRootKey(["*"]);

  const res = await h.get<V1PermissionsGetRoleResponse>({
    url: `/v1/permissions.getRole?roleId=${roleId}`,
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${root.key}`,
    },
  });

  expect(res.status).toEqual(404);
  expect(res.body).toMatchObject({
    error: {
      code: "NOT_FOUND",
      docs: "https://unkey.dev/docs/api-reference/errors/code/NOT_FOUND",
      message: `role ${roleId} not found`,
    },
  });
});
