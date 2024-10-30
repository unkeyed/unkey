import { expect, test } from "vitest";

import { newId } from "@unkey/id";
import { IntegrationHarness } from "src/pkg/testutil/integration-harness";

import type { V1PermissionsGetPermissionResponse } from "./v1_permissions_getPermission";

test("permission does not exist", async (t) => {
  const h = await IntegrationHarness.init(t);
  const permissionId = newId("test");

  const root = await h.createRootKey(["*"]);

  const res = await h.get<V1PermissionsGetPermissionResponse>({
    url: `/v1/permissions.getPermission?permissionId=${permissionId}`,
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
      message: `permission ${permissionId} not found`,
    },
  });
});
