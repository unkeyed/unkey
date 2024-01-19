import { schema } from "@unkey/db";
import { newId } from "@unkey/id";

import { randomUUID } from "node:crypto";
import { describe } from "node:test";
import { Harness } from "@/pkg/testutil/harness";
import { runSharedRoleTests } from "@/pkg/testutil/test_route_roles";
import { expect, test } from "vitest";
import { LegacyApisDeleteApiResponse, registerLegacyApisDeleteApi } from "./legacy_apis_deleteApi";

runSharedRoleTests<LegacyApisDeleteApiResponse>({
  registerHandler: registerLegacyApisDeleteApi,
  prepareRequest: async (h) => {
    const apiId = newId("api");
    await h.db.insert(schema.apis).values({
      id: apiId,
      name: "test",
      workspaceId: h.resources.userWorkspace.id,
    });

    return {
      method: "DELETE",
      url: `/v1/apis/${apiId}`,
    };
  },
});

describe("correct roles", () => {
  test.each([
    { name: "legacy", roles: ["*"] },
    { name: "legacy and more", roles: ["*", randomUUID()] },
    { name: "wildcard", roles: ["api.*.delete_api"] },
    { name: "wildcard and more", roles: ["api.*.delete_api", randomUUID()] },
    { name: "specific apiId", roles: [(apiId: string) => `api.${apiId}.delete_api`] },
    {
      name: "specific apiId and more",
      roles: [(apiId: string) => `api.${apiId}.delete_api`, randomUUID()],
    },
  ])("$name", async ({ roles }) => {
    const h = await Harness.init();
    h.useRoutes(registerLegacyApisDeleteApi);

    const apiId = newId("api");
    await h.db.insert(schema.apis).values({
      id: apiId,
      name: "test",
      workspaceId: h.resources.userWorkspace.id,
    });

    const root = await h.createRootKey(
      roles.map((role) => (typeof role === "string" ? role : role(apiId))),
    );

    const res = await h.delete({
      url: `/v1/apis/${apiId}`,
      headers: {
        Authorization: `Bearer ${root.key}`,
      },
    });
    expect(res.status).toEqual(200);

    const found = await h.resources.database.query.apis.findFirst({
      where: (table, { eq }) => eq(table.id, apiId),
    });
    expect(found).toBeDefined();
    expect(found!.deletedAt).toBeDefined();
  });
});
