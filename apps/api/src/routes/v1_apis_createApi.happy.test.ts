import { afterEach, beforeEach, expect, test } from "vitest";

import { randomUUID } from "crypto";
import { RouteHarness } from "@/pkg/testutil/route-harness";
import {
  V1ApisCreateApiRequest,
  V1ApisCreateApiResponse,
  registerV1ApisCreateApi,
} from "./v1_apis_createApi";

let h: RouteHarness;
beforeEach(async () => {
  h = await RouteHarness.init();
  h.useRoutes(registerV1ApisCreateApi);
  await h.seed();
});
afterEach(async () => {
  await h.teardown();
});
test("creates new api", async () => {
  const root = await h.createRootKey(["api.*.create_api"]);
  const res = await h.post<V1ApisCreateApiRequest, V1ApisCreateApiResponse>({
    url: "/v1/apis.createApi",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${root.key}`,
    },
    body: {
      name: randomUUID(),
    },
  });

  expect(res.status).toEqual(200);

  const found = await h.db.query.apis.findFirst({
    where: (table, { eq }) => eq(table.id, res.body.apiId),
  });
  expect(found).toBeDefined();
});
