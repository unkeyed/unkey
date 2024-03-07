import { afterEach, beforeEach, expect, test } from "vitest";

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
test.each([
  { name: "empty name", apiName: "" },
  { name: "short name", apiName: "ab" },
])("$name", async ({ apiName }) => {
  const { key: rootKey } = await h.createRootKey(["*"]);

  const res = await h.post<V1ApisCreateApiRequest, V1ApisCreateApiResponse>({
    url: "/v1/apis.createApi",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${rootKey}`,
    },
    body: {
      name: apiName,
    },
  });

  expect(res.status).toEqual(400);
  expect(res.body).toMatchObject({
    error: {
      code: "BAD_REQUEST",
      docs: "https://unkey.dev/docs/api-reference/errors/code/BAD_REQUEST",
      message:
        'too_small: name: String must contain at least 3 character(s), See "https://unkey.dev/docs/api-reference" for more details',
    },
  });
});
