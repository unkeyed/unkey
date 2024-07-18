import { expect, test } from "vitest";

import { randomUUID } from "node:crypto";
import { IntegrationHarness } from "src/pkg/testutil/integration-harness";

import type {
  V1PermissionsCreatePermissionRequest,
  V1PermissionsCreatePermissionResponse,
} from "./v1_permissions_createPermission";

test("creates new permission", async (t) => {
  const h = await IntegrationHarness.init(t);
  const root = await h.createRootKey(["rbac.*.create_permission"]);
  const res = await h.post<
    V1PermissionsCreatePermissionRequest,
    V1PermissionsCreatePermissionResponse
  >({
    url: "/v1/permissions.createPermission",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${root.key}`,
    },
    body: {
      name: randomUUID(),
    },
  });

  expect(res.status, `expected 200, received: ${JSON.stringify(res, null, 2)}`).toBe(200);

  const found = await h.db.primary.query.permissions.findFirst({
    where: (table, { eq }) => eq(table.id, res.body.permissionId),
  });
  expect(found).toBeDefined();
});

test("creating the same permission twice does not error", async (t) => {
  const h = await IntegrationHarness.init(t);
  const root = await h.createRootKey(["rbac.*.create_permission"]);

  const name = randomUUID();

  for (let i = 0; i < 2; i++) {
    const res = await h.post<
      V1PermissionsCreatePermissionRequest,
      V1PermissionsCreatePermissionResponse
    >({
      url: "/v1/permissions.createPermission",
      headers: {
        "Content-Type": "application/json",
        Authorization: `Bearer ${root.key}`,
      },
      body: {
        name,
      },
    });

    expect(res.status, `expected 200, received: ${JSON.stringify(res, null, 2)}`).toBe(200);
  }
});
