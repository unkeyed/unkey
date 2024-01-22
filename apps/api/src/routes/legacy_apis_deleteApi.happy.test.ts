import { expect, test } from "vitest";

import { schema } from "@unkey/db";
import { newId } from "@unkey/id";

import { Harness } from "@/pkg/testutil/harness";
import { LegacyApisDeleteApiResponse, registerLegacyApisDeleteApi } from "./legacy_apis_deleteApi";

test("soft deletes api", async () => {
  const h = await Harness.init();
  h.useRoutes(registerLegacyApisDeleteApi);

  const apiId = newId("key");
  await h.db.insert(schema.apis).values({
    id: apiId,
    name: "test",
    workspaceId: h.resources.userWorkspace.id,
  });

  const root = await h.createRootKey([`api.${apiId}.delete_api`]);
  const res = await h.delete<LegacyApisDeleteApiResponse>({
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
  expect(found!.deletedAt!.getTime() - Date.now()).toBeLessThan(10_000); // 10s play
});
