import { newApp } from "@/pkg/hono/app";
import { expect, test } from "bun:test";

import { init } from "@/pkg/global";
import { testEnv } from "@/pkg/testutil/env";
import { fetchRoute } from "@/pkg/testutil/request";
import { seed } from "@/pkg/testutil/seed";
import { LegacyApisDeleteApiResponse, registerLegacyApisDeleteApi } from "./legacy_apis_deleteApi";

test("deletes the api", async () => {
  const env = testEnv();
  // @ts-ignore
  init({ env });

  const r = await seed(env);
  const app = newApp();
  registerLegacyApisDeleteApi(app);

  const res = await fetchRoute<never, LegacyApisDeleteApiResponse>(app, {
    method: "DELETE",
    url: `/v1/apis/${r.userApi.id}`,
    headers: {
      Authorization: `Bearer ${r.rootKey}`,
    },
  });

  expect(res.status).toEqual(200);
  expect(res.body).toEqual({});

  const found = await r.database.query.apis.findFirst({
    where: (table, { eq }) => eq(table.id, r.userApi.id),
  });
  expect(found).toBeUndefined();
});
