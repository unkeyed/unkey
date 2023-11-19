import { newApp } from "@/pkg/hono/app";
import { expect, test } from "bun:test";

import { init } from "@/pkg/global";
import { testEnv } from "@/pkg/testutil/env";
import { fetchRoute } from "@/pkg/testutil/request";
import { seed } from "@/pkg/testutil/seed";
import {
  V1ApisDeleteApiRequest,
  V1ApisDeleteApiResponse,
  registerV1ApisDeleteApi,
} from "./v1_apis_deleteApi";

test("deletes the api", async () => {
  const env = testEnv();
  // @ts-ignore
  init({ env });

  const r = await seed(env);
  const app = newApp();
  registerV1ApisDeleteApi(app);

  const res = await fetchRoute<V1ApisDeleteApiRequest, V1ApisDeleteApiResponse>(app, {
    method: "POST",
    url: "/v1/apis.deleteApi",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${r.rootKey}`,
    },
    body: {
      apiId: r.userApi.id,
    },
  });
  console.log(res);

  expect(res.status).toEqual(200);
  expect(res.body).toEqual({});

  const found = await r.database.query.apis.findFirst({
    where: (table, { eq }) => eq(table.id, r.userApi.id),
  });
  expect(found).toBeUndefined();
});
