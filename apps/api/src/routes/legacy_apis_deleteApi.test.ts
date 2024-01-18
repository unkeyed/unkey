import { Harness } from "@/pkg/testutil/harness";
import { expect, test } from "vitest";
import { LegacyApisDeleteApiResponse, registerLegacyApisDeleteApi } from "./legacy_apis_deleteApi";

test("deletes the api", async () => {
  const h = await Harness.init();
  h.useRoutes(registerLegacyApisDeleteApi);

  const res = await h.delete<LegacyApisDeleteApiResponse>({
    url: `/v1/apis/${h.resources.userApi.id}`,
    headers: {
      Authorization: `Bearer ${h.resources.rootKey}`,
    },
  });

  expect(res.status).toEqual(200);
  expect(res.body).toEqual({});

  const found = await h.resources.database.query.apis.findFirst({
    where: (table, { eq, and, isNull }) =>
      and(eq(table.id, h.resources.userApi.id), isNull(table.deletedAt)),
  });
  expect(found).toBeUndefined();
});
