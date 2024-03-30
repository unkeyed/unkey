import { expect, test } from "vitest";

import { randomUUID } from "node:crypto";
import { RouteHarness } from "src/pkg/testutil/route-harness";
import type { V1ApisCreateApiRequest, V1ApisCreateApiResponse } from "./v1_apis_createApi";

test("creates new api", async (t) => {
  const h = await RouteHarness.init(t);
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
