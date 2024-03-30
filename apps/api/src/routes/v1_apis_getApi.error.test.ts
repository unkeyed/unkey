import { expect, test } from "vitest";

import { newId } from "@unkey/id";
import { RouteHarness } from "src/pkg/testutil/route-harness";
import type { V1ApisGetApiResponse } from "./v1_apis_getApi";

test("api does not exist", async (t) => {
  const h = await RouteHarness.init(t);
  const apiId = newId("api");

  const root = await h.createRootKey(["*"]);

  const res = await h.get<V1ApisGetApiResponse>({
    url: `/v1/apis.getApi?apiId=${apiId}`,
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
      message: `api ${apiId} not found`,
    },
  });
});
